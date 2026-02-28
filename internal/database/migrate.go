package database

import (
	"context"
	"embed"
	"fmt"
	"log"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// RunMigrations applies all pending migrations
func RunMigrations(ctx context.Context) error {
	if pool == nil {
		return fmt.Errorf("database not connected")
	}

	// Create migrations table if not exists
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INT PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current version
	currentVersion, err := GetMigrationVersion(ctx)
	if err != nil {
		return err
	}

	// Read migration files
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		log.Println("[Migrate] No migrations directory found")
		return nil
	}

	// Sort and apply migrations
	var migrations []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".up.sql") {
			migrations = append(migrations, entry.Name())
		}
	}
	sort.Strings(migrations)

	applied := 0
	for _, filename := range migrations {
		// Parse version from filename (e.g., "001_create_users.up.sql")
		var version int
		_, err := fmt.Sscanf(filename, "%d_", &version)
		if err != nil {
			continue
		}

		if version <= currentVersion {
			continue
		}

		// Read and execute migration
		content, err := migrationFS.ReadFile("migrations/" + filename)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", filename, err)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}

		_, err = tx.Exec(ctx, string(content))
		if err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("failed to execute migration %s: %w", filename, err)
		}

		_, err = tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version)
		if err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("failed to record migration %s: %w", filename, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", filename, err)
		}

		log.Printf("[Migrate] Applied migration %s", filename)
		applied++
	}

	if applied > 0 {
		log.Printf("[Migrate] Applied %d migrations", applied)
	} else {
		log.Println("[Migrate] No new migrations to apply")
	}

	return nil
}

// GetMigrationVersion returns the current schema version
func GetMigrationVersion(ctx context.Context) (int, error) {
	if pool == nil {
		return 0, fmt.Errorf("database not connected")
	}

	var version int
	err := pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(version), 0) FROM schema_migrations
	`).Scan(&version)

	if err != nil {
		// Table might not exist yet
		if strings.Contains(err.Error(), "does not exist") {
			return 0, nil
		}
		return 0, err
	}

	return version, nil
}
