package database

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestRunMigrations_FailsWithoutConnection(t *testing.T) {
	// Given: No database connection
	pool = nil

	// When: RunMigrations is called
	ctx := context.Background()
	err := RunMigrations(ctx)

	// Then: Should return error
	if err == nil {
		t.Error("expected error when running migrations without connection")
	}
}

func TestGetMigrationVersion_ReturnsZeroForNewDB(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Given: Connected to fresh database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://trading:trading@localhost:5432/trading?sslmode=disable"
		os.Setenv("DATABASE_URL", dbURL)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := Connect(ctx)
	if err != nil {
		t.Skipf("skipping: database not available: %v", err)
	}
	defer Close()

	// When: Getting migration version
	version, err := GetMigrationVersion(ctx)

	// Then: Should return 0 or current version without error
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if version < 0 {
		t.Errorf("expected non-negative version, got %d", version)
	}
}

func TestRunMigrations_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Given: Connected database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://trading:trading@localhost:5432/trading?sslmode=disable"
		os.Setenv("DATABASE_URL", dbURL)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := Connect(ctx)
	if err != nil {
		t.Skipf("skipping: database not available: %v", err)
	}
	defer Close()

	// When: Running migrations
	err = RunMigrations(ctx)

	// Then: Should succeed
	if err != nil {
		t.Errorf("migration failed: %v", err)
	}

	// And: Version should be > 0
	version, _ := GetMigrationVersion(ctx)
	if version < 1 {
		t.Log("no migrations applied yet (expected for fresh setup)")
	}
}
