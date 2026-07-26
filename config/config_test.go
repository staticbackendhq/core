package config

import "testing"

func TestLoadConfigPostgresPoolDefaults(t *testing.T) {
	t.Setenv("POSTGRES_MAX_OPEN_CONNS", "")
	t.Setenv("POSTGRES_MAX_IDLE_CONNS", "")
	t.Setenv("POSTGRES_CONN_MAX_LIFETIME_SECONDS", "")
	t.Setenv("POSTGRES_CONN_MAX_IDLE_TIME_SECONDS", "")

	cfg := LoadConfig()
	if cfg.PostgresMaxOpenConns != defaultPostgresMaxOpenConns {
		t.Fatalf("max open connections = %d, want %d", cfg.PostgresMaxOpenConns, defaultPostgresMaxOpenConns)
	}
	if cfg.PostgresMaxIdleConns != defaultPostgresMaxIdleConns {
		t.Fatalf("max idle connections = %d, want %d", cfg.PostgresMaxIdleConns, defaultPostgresMaxIdleConns)
	}
	if cfg.PostgresConnMaxLifetimeSeconds != defaultPostgresConnMaxLifetimeSeconds {
		t.Fatalf("max lifetime = %d, want %d", cfg.PostgresConnMaxLifetimeSeconds, defaultPostgresConnMaxLifetimeSeconds)
	}
	if cfg.PostgresConnMaxIdleTimeSeconds != defaultPostgresConnMaxIdleTimeSeconds {
		t.Fatalf("max idle time = %d, want %d", cfg.PostgresConnMaxIdleTimeSeconds, defaultPostgresConnMaxIdleTimeSeconds)
	}
}

func TestLoadConfigPostgresPoolValues(t *testing.T) {
	t.Setenv("POSTGRES_MAX_OPEN_CONNS", "7")
	t.Setenv("POSTGRES_MAX_IDLE_CONNS", "3")
	t.Setenv("POSTGRES_CONN_MAX_LIFETIME_SECONDS", "60")
	t.Setenv("POSTGRES_CONN_MAX_IDLE_TIME_SECONDS", "15")

	cfg := LoadConfig()
	if cfg.PostgresMaxOpenConns != 7 || cfg.PostgresMaxIdleConns != 3 ||
		cfg.PostgresConnMaxLifetimeSeconds != 60 || cfg.PostgresConnMaxIdleTimeSeconds != 15 {
		t.Fatalf("unexpected PostgreSQL pool config: %+v", cfg)
	}
}

func TestLoadConfigPostgresPoolInvalidValuesUseDefaults(t *testing.T) {
	t.Setenv("POSTGRES_MAX_OPEN_CONNS", "invalid")
	t.Setenv("POSTGRES_MAX_IDLE_CONNS", "0")
	t.Setenv("POSTGRES_CONN_MAX_LIFETIME_SECONDS", "-1")
	t.Setenv("POSTGRES_CONN_MAX_IDLE_TIME_SECONDS", "invalid")

	cfg := LoadConfig()
	if cfg.PostgresMaxOpenConns != defaultPostgresMaxOpenConns ||
		cfg.PostgresMaxIdleConns != defaultPostgresMaxIdleConns ||
		cfg.PostgresConnMaxLifetimeSeconds != defaultPostgresConnMaxLifetimeSeconds ||
		cfg.PostgresConnMaxIdleTimeSeconds != defaultPostgresConnMaxIdleTimeSeconds {
		t.Fatalf("invalid pool values were not normalized: %+v", cfg)
	}
}
