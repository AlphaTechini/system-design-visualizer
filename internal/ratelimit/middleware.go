package ratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/AlphaTechini/system-design-visualizer/internal/database"
	"github.com/jackc/pgx/v5"
)

// RateLimitMiddleware checks IP-based generation limits
type RateLimitMiddleware struct {
	db *database.SupabaseClient
}

// RateLimitInfo represents current usage for an IP
type RateLimitInfo struct {
	FreeRemaining       int       `json:"free_remaining"`
	BonusRemaining      int       `json:"bonus_remaining"`
	ResetAt             time.Time `json:"reset_at"`
	UpgradeURL          string    `json:"upgrade_url"`
	TotalGenerationsToday int     `json:"total_generations_today"`
}

// NewRateLimitMiddleware creates middleware with database connection
func NewRateLimitMiddleware(db *database.SupabaseClient) *RateLimitMiddleware {
	return &RateLimitMiddleware{db: db}
}

// Middleware wraps HTTP handlers with rate limiting
func (rl *RateLimitMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		
		// Skip rate limit for health checks
		if r.URL.Path == "/health" || r.URL.Path == "/api/v1/rate-limit" {
			next.ServeHTTP(w, r)
			return
		}
		
		// Check and increment rate limit
		info, allowed, err := rl.checkAndIncrement(r.Context(), ip)
		if err != nil {
			http.Error(w, fmt.Sprintf("rate limit error: %v", err), http.StatusInternalServerError)
			return
		}
		
		// Add rate limit info to response headers
		w.Header().Set("X-RateLimit-Free-Remaining", fmt.Sprintf("%d", info.FreeRemaining))
		w.Header().Set("X-RateLimit-Bonus-Remaining", fmt.Sprintf("%d", info.BonusRemaining))
		w.Header().Set("X-RateLimit-Reset", info.ResetAt.Format(time.RFC3339))
		
		if !allowed {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":         "daily_limit_reached",
				"message":       "You've reached your free daily limit",
				"free_remaining": 0,
				"bonus_remaining": info.BonusRemaining,
				"upgrade_url":     "/pricing",
				"retry_after":     info.ResetAt.Format(time.RFC3339),
			})
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// checkAndIncrement verifies limit and increments counter atomically
func (rl *RateLimitMiddleware) checkAndIncrement(ctx context.Context, ip string) (*RateLimitInfo, bool, error) {
	// Use a transaction for atomic read-modify-write
	tx, err := rl.db.Pool().Begin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	
	today := time.Now().Truncate(24 * time.Hour)
	
	// Try to get existing record
	var freeUsed, bonusUsed int
	var lastReset time.Time
	
	err = tx.QueryRow(ctx, `
		SELECT free_generations_used, bonus_generations_used, last_reset_date
		FROM rate_limits
		WHERE ip_address = $1
	`, ip).Scan(&freeUsed, &bonusUsed, &lastReset)
	
	if err == pgx.ErrNoRows {
		// Create new record
		_, err = tx.Exec(ctx, `
			INSERT INTO rate_limits (ip_address, free_generations_used, bonus_generations_used, last_reset_date)
			VALUES ($1, 0, 0, $2)
		`, ip, today)
		if err != nil {
			return nil, false, fmt.Errorf("insert rate limit: %w", err)
		}
		freeUsed = 0
		bonusUsed = 0
		lastReset = today
	} else if err != nil {
		return nil, false, fmt.Errorf("query rate limit: %w", err)
	}
	
	// Reset counters if new day
	if lastReset.Before(today) {
		_, err = tx.Exec(ctx, `
			UPDATE rate_limits
			SET free_generations_used = 0, bonus_generations_used = 0, last_reset_date = $2
			WHERE ip_address = $1
		`, ip, today)
		if err != nil {
			return nil, false, fmt.Errorf("reset counters: %w", err)
		}
		freeUsed = 0
		bonusUsed = 0
	}
	
	// Check if limit reached
	allowed := true
	if freeUsed >= 1 && bonusUsed >= 1 {
		allowed = false
	}
	
	// Increment counter (use free first, then bonus)
	if allowed {
		if freeUsed < 1 {
			_, err = tx.Exec(ctx, `
				UPDATE rate_limits SET free_generations_used = free_generations_used + 1 WHERE ip_address = $1
			`, ip)
		} else {
			_, err = tx.Exec(ctx, `
				UPDATE rate_limits SET bonus_generations_used = bonus_generations_used + 1 WHERE ip_address = $1
			`, ip)
		}
		if err != nil {
			return nil, false, fmt.Errorf("increment counter: %w", err)
		}
		
		// Update local vars for response
		if freeUsed < 1 {
			freeUsed++
		} else {
			bonusUsed++
		}
	}
	
	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf("commit transaction: %w", err)
	}
	
	// Build response info
	resetAt := today.Add(24 * time.Hour)
	
	info := &RateLimitInfo{
		FreeRemaining:       max(0, 1-freeUsed),
		BonusRemaining:      max(0, 1-bonusUsed),
		ResetAt:             resetAt,
		UpgradeURL:          "/pricing",
		TotalGenerationsToday: freeUsed + bonusUsed,
	}
	
	return info, allowed, nil
}

// GetInfo returns current rate limit status without incrementing
func (rl *RateLimitMiddleware) GetInfo(ctx context.Context, ip string) (*RateLimitInfo, error) {
	today := time.Now().Truncate(24 * time.Hour)
	
	var freeUsed, bonusUsed int
	var lastReset time.Time
	
	err := rl.db.Pool().QueryRow(ctx, `
		SELECT free_generations_used, bonus_generations_used, last_reset_date
		FROM rate_limits
		WHERE ip_address = $1
	`, ip).Scan(&freeUsed, &bonusUsed, &lastReset)
	
	if err == pgx.ErrNoRows {
		return &RateLimitInfo{
			FreeRemaining:       1,
			BonusRemaining:      1,
			ResetAt:             today.Add(24 * time.Hour),
			UpgradeURL:          "/pricing",
			TotalGenerationsToday: 0,
		}, nil
	}
	
	if err != nil {
		return nil, fmt.Errorf("query rate limit: %w", err)
	}
	
	// Reset if new day
	if lastReset.Before(today) {
		freeUsed = 0
		bonusUsed = 0
	}
	
	resetAt := today.Add(24 * time.Hour)
	
	return &RateLimitInfo{
		FreeRemaining:       max(0, 1-freeUsed),
		BonusRemaining:      max(0, 1-bonusUsed),
		ResetAt:             resetAt,
		UpgradeURL:          "/pricing",
		TotalGenerationsToday: freeUsed + bonusUsed,
	}, nil
}

// GetClientIP extracts real IP from request (handles proxies) - exported for handlers
func GetClientIP(r *http.Request) string {
	return getClientIP(r)
}

// getClientIP extracts real IP from request (handles proxies)
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (proxy/load balancer)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take first IP in chain
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// Fallback to RemoteAddr
	return r.RemoteAddr
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
