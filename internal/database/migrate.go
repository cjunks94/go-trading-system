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

	if err := ensureMigrationsTable(ctx); err != nil {
		return err
	}

	currentVersion, err := GetMigrationVersion(ctx)
	if err != nil {
		return err
	}

	migrations, err := getMigrationFiles()
	if err != nil {
		return nil // No migrations directory is not an error
	}

	applied := 0
	for _, filename := range migrations {
		version := parseVersion(filename)
		if version <= currentVersion {
			continue
		}

		if err := applyMigration(ctx, filename, version); err != nil {
			return err
		}
		applied++
	}

	logMigrationResult(applied)
	return nil
}

// ensureMigrationsTable creates the schema_migrations table if needed
func ensureMigrationsTable(ctx context.Context) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INT PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}
	return nil
}

// getMigrationFiles reads and sorts migration files
func getMigrationFiles() ([]string, error) {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		log.Println("[Migrate] No migrations directory found")
		return nil, err
	}

	var migrations []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".up.sql") {
			migrations = append(migrations, entry.Name())
		}
	}
	sort.Strings(migrations)
	return migrations, nil
}

// parseVersion extracts version number from filename (e.g., "001_create_users.up.sql")
func parseVersion(filename string) int {
	var version int
	_, _ = fmt.Sscanf(filename, "%d_", &version)
	return version
}

// applyMigration executes a single migration in a transaction
func applyMigration(ctx context.Context, filename string, version int) error {
	content, err := migrationFS.ReadFile("migrations/" + filename)
	if err != nil {
		return fmt.Errorf("failed to read migration %s: %w", filename, err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err.Error() != "tx is closed" {
			log.Printf("[Migrate] Rollback warning: %v", err)
		}
	}()

	if _, err = tx.Exec(ctx, string(content)); err != nil {
		return fmt.Errorf("failed to execute migration %s: %w", filename, err)
	}

	if _, err = tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
		return fmt.Errorf("failed to record migration %s: %w", filename, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit migration %s: %w", filename, err)
	}

	log.Printf("[Migrate] Applied migration %s", filename)
	return nil
}

// logMigrationResult logs the migration summary
func logMigrationResult(applied int) {
	if applied > 0 {
		log.Printf("[Migrate] Applied %d migrations", applied)
	} else {
		log.Println("[Migrate] No new migrations to apply")
	}
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
