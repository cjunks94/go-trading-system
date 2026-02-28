package database

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestConnect_RequiresDatabaseURL(t *testing.T) {
	// Given: No DATABASE_URL set
	os.Unsetenv("DATABASE_URL")

	// When: Connect is called
	ctx := context.Background()
	_, err := Connect(ctx)

	// Then: Error should indicate missing URL
	if err == nil {
		t.Fatal("expected error when DATABASE_URL is not set")
	}
	if err.Error() != "DATABASE_URL environment variable is required" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestConnect_InvalidURL(t *testing.T) {
	// Given: Invalid DATABASE_URL
	os.Setenv("DATABASE_URL", "not-a-valid-url")
	defer os.Unsetenv("DATABASE_URL")

	// When: Connect is called
	ctx := context.Background()
	_, err := Connect(ctx)

	// Then: Error should indicate parse failure
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestGetPool_ReturnsNilBeforeConnect(t *testing.T) {
	// Given: No connection established
	pool = nil

	// When: GetPool is called
	p := GetPool()

	// Then: Should return nil
	if p != nil {
		t.Error("expected nil pool before Connect")
	}
}

func TestPing_FailsWhenNotConnected(t *testing.T) {
	// Given: No connection
	pool = nil

	// When: Ping is called
	ctx := context.Background()
	err := Ping(ctx)

	// Then: Should return error
	if err == nil {
		t.Error("expected error when pinging without connection")
	}
}

func TestHealthCheck_ReturnsUnhealthyWhenNotConnected(t *testing.T) {
	// Given: No connection
	pool = nil

	// When: HealthCheck is called
	ctx := context.Background()
	status := HealthCheck(ctx)

	// Then: Status should be unhealthy
	if status["status"] != "unhealthy" {
		t.Errorf("expected unhealthy status, got %v", status["status"])
	}
}

// Integration tests - require running PostgreSQL
func TestConnect_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Given: Valid DATABASE_URL (from docker-compose)
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://trading:trading@localhost:5432/trading?sslmode=disable"
		os.Setenv("DATABASE_URL", dbURL)
	}

	// When: Connect is called
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	p, err := Connect(ctx)

	// Then: Connection should succeed
	if err != nil {
		t.Skipf("skipping: database not available: %v", err)
	}
	defer Close()

	if p == nil {
		t.Fatal("expected non-nil pool")
	}

	// And: Ping should work
	if err := Ping(ctx); err != nil {
		t.Errorf("ping failed: %v", err)
	}

	// And: HealthCheck should be healthy
	status := HealthCheck(ctx)
	if status["status"] != "healthy" {
		t.Errorf("expected healthy status, got %v", status)
	}
}
