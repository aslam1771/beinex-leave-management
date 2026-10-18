package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"beinex-leave-management/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps the pgxpool.Pool connection.
type DB struct {
	Pool *pgxpool.Pool
}

// NewPostgresPool establishes a connection pool to PostgreSQL using pgx.
func NewPostgresPool(ctx context.Context, cfg *config.Config) (*DB, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	// Apply connection pool tuning from config
	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime

	// Connection and ping timeouts (generous to accommodate Neon serverless cold starts)
	connectCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connectivity
	pingCtx, pingCancel := context.WithTimeout(ctx, 10*time.Second)
	defer pingCancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("[database] successfully connected to PostgreSQL at %s", cfg.SanitizedDatabaseURL())
	return &DB{Pool: pool}, nil
}

// Ping checks if the database is reachable.
func (db *DB) Ping(ctx context.Context) error {
	if db == nil || db.Pool == nil {
		return fmt.Errorf("database connection pool is not initialized")
	}
	return db.Pool.Ping(ctx)
}

// Close gracefully closes the connection pool.
func (db *DB) Close() {
	if db != nil && db.Pool != nil {
		db.Pool.Close()
		log.Println("[database] connection pool closed")
	}
}
