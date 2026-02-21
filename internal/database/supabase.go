package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SupabaseClient wraps PostgreSQL connection (Supabase uses standard Postgres)
type SupabaseClient struct {
	pool *pgxpool.Pool
}

// Config holds database configuration
type Config struct {
	Host        string // e.g., "xyz.supabase.co"
	Port        int    // typically 5432
	Database    string // typically "postgres"
	User        string // typically "postgres"
	Password    string // from Supabase dashboard
	SSLMode     string // "require" for Supabase
	MaxConns    int32  // connection pool size
	MinConns    int32
}

// NewSupabaseClient creates database connection pool
func NewSupabaseClient(ctx context.Context, cfg Config) (*SupabaseClient, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Database, cfg.User, cfg.Password, cfg.SSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Connection pool settings
	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	// Health check before returning
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	client := &SupabaseClient{pool: pool}

	// Run migrations
	if err := client.runMigrations(ctx); err != nil {
		return nil, fmt.Errorf("migrations: %w", err)
	}

	return client, nil
}

// Close closes the connection pool
func (c *SupabaseClient) Close() {
	c.pool.Close()
}

// runMigrations creates tables if they don't exist
func (c *SupabaseClient) runMigrations(ctx context.Context) error {
	migrations := []string{
		// Rate limiting by IP
		`CREATE TABLE IF NOT EXISTS rate_limits (
			ip_address INET PRIMARY KEY,
			free_generations_used INT DEFAULT 0,
			bonus_generations_used INT DEFAULT 0,
			last_reset_date DATE DEFAULT CURRENT_DATE,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,

		// Design sessions
		`CREATE TABLE IF NOT EXISTS designs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			ip_address INET NOT NULL,
			name VARCHAR(255) NOT NULL,
			requirements_json JSONB NOT NULL,
			ai_recommendations_json JSONB,
			mermaid_code TEXT,
			terraform_code TEXT,
			cost_estimate_json JSONB,
			status VARCHAR(50) DEFAULT 'draft',
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,

		// Cached AI responses
		`CREATE TABLE IF NOT EXISTS ai_cache (
			requirements_hash VARCHAR(64) PRIMARY KEY,
			ai_response_json JSONB NOT NULL,
			hit_count INT DEFAULT 1,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			last_used_at TIMESTAMPTZ DEFAULT NOW()
		)`,

		// Indexes for performance
		`CREATE INDEX IF NOT EXISTS idx_designs_ip ON designs(ip_address)`,
		`CREATE INDEX IF NOT EXISTS idx_designs_status ON designs(status)`,
		`CREATE INDEX IF NOT EXISTS idx_designs_created ON designs(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_cache_last_used ON ai_cache(last_used_at DESC)`,

		// Trigger to update updated_at timestamp
		`CREATE OR REPLACE FUNCTION update_updated_at_column()
			RETURNS TRIGGER AS $$
			BEGIN
				NEW.updated_at = NOW();
				RETURN NEW;
			END;
			$$ language 'plpgsql'`,

		`DROP TRIGGER IF EXISTS update_designs_updated_at ON designs`,
		`CREATE TRIGGER update_designs_updated_at
			BEFORE UPDATE ON designs
			FOR EACH ROW
			EXECUTE FUNCTION update_updated_at_column()`,

		`DROP TRIGGER IF EXISTS update_rate_limits_updated_at ON rate_limits`,
		`CREATE TRIGGER update_rate_limits_updated_at
			BEFORE UPDATE ON rate_limits
			FOR EACH ROW
			EXECUTE FUNCTION update_updated_at_column()`,
	}

	// Run each migration
	for i, migration := range migrations {
		if _, err := c.pool.Exec(ctx, migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", i, err)
		}
	}

	return nil
}

// Pool returns the underlying connection pool (for advanced queries)
func (c *SupabaseClient) Pool() *pgxpool.Pool {
	return c.pool
}
