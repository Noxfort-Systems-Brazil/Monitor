// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// File: internal/storage/device_repo_test.go

package storage

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	cleanup := func() {
		_ = db.Close()
	}

	return db, cleanup
}

func TestDeviceRepository_DebouncedHeartbeat(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewDeviceRepository(db)
	repo.SetDebounceInterval(500 * time.Millisecond)

	now := time.Now().Truncate(time.Second)

	// 1. First heartbeat: must persist immediately
	err := repo.UpdateLastSeen("MACHINE-01", now)
	if err != nil {
		t.Fatalf("UpdateLastSeen failed: %v", err)
	}

	devices, err := repo.GetAllDevices()
	if err != nil {
		t.Fatalf("GetAllDevices failed: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("Expected 1 device, got %d", len(devices))
	}
	if devices[0].Identifier != "MACHINE-01" {
		t.Errorf("Expected identifier MACHINE-01, got %s", devices[0].Identifier)
	}

	// 2. Second heartbeat within debounce interval (100ms later)
	secondTime := now.Add(100 * time.Millisecond)
	err = repo.UpdateLastSeen("MACHINE-01", secondTime)
	if err != nil {
		t.Fatalf("Second UpdateLastSeen failed: %v", err)
	}

	// GetAllDevices should reflect secondTime thanks to the in-memory cache overlay
	devices, err = repo.GetAllDevices()
	if err != nil {
		t.Fatalf("GetAllDevices failed: %v", err)
	}
	if !devices[0].LastSeen.Equal(secondTime) {
		t.Errorf("Expected LastSeen overlay %v, got %v", secondTime, devices[0].LastSeen)
	}

	// 3. Third heartbeat after debounce interval has passed
	time.Sleep(550 * time.Millisecond)
	thirdTime := secondTime.Add(600 * time.Millisecond)
	err = repo.UpdateLastSeen("MACHINE-01", thirdTime)
	if err != nil {
		t.Fatalf("Third UpdateLastSeen failed: %v", err)
	}

	devices, err = repo.GetAllDevices()
	if err != nil {
		t.Fatalf("GetAllDevices failed: %v", err)
	}
	if !devices[0].LastSeen.Equal(thirdTime) {
		t.Errorf("Expected LastSeen %v, got %v", thirdTime, devices[0].LastSeen)
	}

	// 4. ForceUpdateLastSeen should always write directly
	forcedTime := thirdTime.Add(1 * time.Second)
	err = repo.ForceUpdateLastSeen("MACHINE-01", forcedTime)
	if err != nil {
		t.Fatalf("ForceUpdateLastSeen failed: %v", err)
	}

	// 5. Delete device
	err = repo.DeleteDevice("MACHINE-01")
	if err != nil {
		t.Fatalf("DeleteDevice failed: %v", err)
	}

	devices, err = repo.GetAllDevices()
	if err != nil {
		t.Fatalf("GetAllDevices failed after deletion: %v", err)
	}
	if len(devices) != 0 {
		t.Errorf("Expected 0 devices after deletion, got %d", len(devices))
	}
}
