package backend

import (
	"database/sql"
	"testing"

	"github.com/staticbackendhq/core/config"
)

func TestNormalizedPostgresPoolConfig(t *testing.T) {
	pool := normalizedPostgresPoolConfig(config.AppConfig{
		PostgresMaxOpenConns:           4,
		PostgresMaxIdleConns:           9,
		PostgresConnMaxLifetimeSeconds: 60,
		PostgresConnMaxIdleTimeSeconds: 10,
	})

	if pool.maxOpenConns != 4 || pool.maxIdleConns != 4 ||
		pool.maxLifetimeSeconds != 60 || pool.maxIdleTimeSeconds != 10 {
		t.Fatalf("unexpected pool config: %+v", pool)
	}
}

func TestConfigurePostgresPoolUsesSafeDefaults(t *testing.T) {
	db, err := sql.Open("postgres", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	pool := configurePostgresPool(db, config.AppConfig{})
	if pool.maxOpenConns != defaultPostgresMaxOpenConns || pool.maxIdleConns != defaultPostgresMaxIdleConns {
		t.Fatalf("unexpected default pool config: %+v", pool)
	}
	if got := db.Stats().MaxOpenConnections; got != defaultPostgresMaxOpenConns {
		t.Fatalf("max open connections = %d, want %d", got, defaultPostgresMaxOpenConns)
	}
}
