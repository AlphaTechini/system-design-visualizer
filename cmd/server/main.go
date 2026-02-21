package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AlphaTechini/system-design-visualizer/internal/database"
	"github.com/AlphaTechini/system-design-visualizer/internal/ratelimit"
	"github.com/gorilla/mux"
)

func main() {
	log.Println("🚀 Starting System Design Visualizer...")

	// Load configuration from environment
	cfg := database.Config{
		Host:     getEnv("SUPABASE_HOST", "required"),
		Port:     5432,
		Database: getEnv("SUPABASE_DATABASE", "postgres"),
		User:     getEnv("SUPABASE_USER", "postgres"),
		Password: getEnv("SUPABASE_PASSWORD", "required"),
		SSLMode:  getEnv("SUPABASE_SSLMODE", "require"),
		MaxConns: 10,
		MinConns: 2,
	}

	// Validate required config
	if cfg.Host == "required" || cfg.Password == "required" {
		log.Fatal("❌ Missing required environment variables: SUPABASE_HOST, SUPABASE_PASSWORD")
	}

	// Initialize database
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.NewSupabaseClient(ctx, cfg)
	if err != nil {
		log.Fatalf("❌ Failed to connect to Supabase: %v", err)
	}
	defer db.Close()

	log.Println("✅ Connected to Supabase PostgreSQL")

	// Initialize rate limiting middleware
	rateLimitMW := ratelimit.NewRateLimitMiddleware(db)

	// Setup HTTP router
	r := mux.NewRouter()

	// Health check endpoint (no rate limit)
	r.HandleFunc("/health", healthHandler).Methods("GET")

	// API routes with rate limiting
	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(rateLimitMW.Middleware)

	// Rate limit status endpoint
	api.HandleFunc("/rate-limit", getRateLimitHandler(rateLimitMW)).Methods("GET")

	// Placeholder for future endpoints
	api.HandleFunc("/designs", createDesignHandler(db)).Methods("POST")
	api.HandleFunc("/designs/{id}", getDesignHandler(db)).Methods("GET")

	// Serve static files (SvelteKit build)
	fs := http.FileServer(http.Dir("../web/static"))
	r.PathPrefix("/").Handler(fs)

	// Start server
	port := getEnv("PORT", "8080")
	addr := fmt.Sprintf(":%s", port)

	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		log.Printf("🌐 Server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("\n🛑 Shutting down server...")

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("❌ Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server stopped gracefully")
}

// Health check handler
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "0.1.0",
	})
}

// Get current rate limit status
func getRateLimitHandler(rlmw *ratelimit.RateLimitMiddleware) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := ratelimit.GetClientIP(r)
		
		info, err := rlmw.GetInfo(r.Context(), ip)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	}
}

// Placeholder handlers
func createDesignHandler(db *database.SupabaseClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement design creation
		w.WriteHeader(http.StatusNotImplemented)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "not_implemented",
		})
	}
}

func getDesignHandler(db *database.SupabaseClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement design retrieval
		w.WriteHeader(http.StatusNotImplemented)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "not_implemented",
		})
	}
}

// Helper: Get environment variable with default
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
