// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems

package storage

import (
	"path/filepath"
	"testing"

	"noxfort-monitor-server/internal/domain"
)

func TestDBManager_SQLite(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	cfg := domain.DatabaseConfig{
		Type:     "sqlite",
		FilePath: dbPath,
	}

	db, driver, err := OpenConnection(cfg)
	if err != nil {
		t.Fatalf("OpenConnection failed: %v", err)
	}
	defer db.Close()

	if driver != "sqlite" {
		t.Errorf("expected driver 'sqlite', got '%s'", driver)
	}

	if err := initSchema(db); err != nil {
		t.Fatalf("initSchema failed: %v", err)
	}

	mgr := NewDBManager(db, driver, cfg)
	status := mgr.GetStatus()
	if !status.Connected {
		t.Errorf("expected status.Connected to be true")
	}

	// Test repository hot-reload
	devRepo := NewDeviceRepository(db)
	mgr.RegisterRepository(devRepo)

	// Create another SQLite DB to switch to
	dbPath2 := filepath.Join(tempDir, "test2.db")
	cfg2 := domain.DatabaseConfig{
		Type:     "sqlite",
		FilePath: dbPath2,
	}

	if err := mgr.Switch(cfg2, true); err != nil {
		t.Fatalf("Switch failed: %v", err)
	}

	if mgr.GetDriver() != "sqlite" {
		t.Errorf("expected driver 'sqlite' after switch")
	}
}

func TestDBManager_Postgres_Local(t *testing.T) {
	// Only run if Postgres is reachable
	cfg := domain.DatabaseConfig{
		Type:     "postgres",
		Host:     "localhost",
		Port:     5432,
		User:     "user_synapse",
		Password: "synapse123",
		DBName:   "banco_de_dados_noxfort",
		Schema:   "schema_monitor",
		SSLMode:  "disable",
	}

	testDB, _, err := OpenConnection(cfg)
	if err != nil {
		t.Skipf("Postgres not accessible with test credentials: %v", err)
		return
	}
	defer testDB.Close()

	// Initialize schema_monitor
	if err := InitPostgresSchema(testDB, cfg.Schema); err != nil {
		t.Fatalf("InitPostgresSchema failed: %v", err)
	}

	exists, err := CheckPostgresSchemaExists(testDB, cfg.Schema)
	if err != nil {
		t.Fatalf("CheckPostgresSchemaExists failed: %v", err)
	}
	if !exists {
		t.Fatalf("Expected schema '%s' to exist after InitPostgresSchema", cfg.Schema)
	}
}
