package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var pool *pgxpool.Pool

// Connect establishes a connection pool to PostgreSQL
func Connect(ctx context.Context) (*pgxpool.Pool, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Connection pool settings
	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	config.HealthCheckPeriod = time.Minute

	// Connect with retry logic
	var p *pgxpool.Pool
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		p, err = pgxpool.NewWithConfig(ctx, config)
		if err == nil {
			// Test the connection
			if pingErr := p.Ping(ctx); pingErr == nil {
				break
			}
			p.Close()
		}

		log.Printf("[Database] Connection attempt %d/%d failed: %v", i+1, maxRetries, err)
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
	}

	pool = p
	log.Println("[Database] Connected to PostgreSQL")
	return p, nil
}

// GetPool returns the connection pool
func GetPool() *pgxpool.Pool {
	return pool
}

// Close closes the connection pool
func Close() {
	if pool != nil {
		pool.Close()
		log.Println("[Database] Connection pool closed")
	}
}

// Ping checks database connectivity
func Ping(ctx context.Context) error {
	if pool == nil {
		return fmt.Errorf("database not connected")
	}
	return pool.Ping(ctx)
}

// HealthCheck returns database health status
func HealthCheck(ctx context.Context) map[string]interface{} {
	status := map[string]interface{}{
		"status": "unhealthy",
	}

	if pool == nil {
		status["error"] = "not connected"
		return status
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		status["error"] = err.Error()
		return status
	}

	stats := pool.Stat()
	status["status"] = "healthy"
	status["total_conns"] = stats.TotalConns()
	status["idle_conns"] = stats.IdleConns()
	status["acquired_conns"] = stats.AcquiredConns()

	return status
}
