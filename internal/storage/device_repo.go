// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: internal/storage/device_repo.go
// Author: Gabriel Moraes
// Date: 2026-01-19
// Modified: 2026-09-04 (PostgreSQL & Hot-Reload Support)

package storage

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"noxfort-monitor-server/internal/domain"
)

// DeviceRepository manages the lifecycle of monitored systems in the database.
type DeviceRepository struct {
	mu               sync.RWMutex
	db               *sql.DB
	driver           string
	debounceMu       sync.Mutex
	lastPersisted    map[string]time.Time
	debounceInterval time.Duration
}

// NewDeviceRepository initializes the repository.
func NewDeviceRepository(db *sql.DB) *DeviceRepository {
	return &DeviceRepository{
		db:               db,
		driver:           "sqlite",
		lastPersisted:    make(map[string]time.Time),
		debounceInterval: 10 * time.Second,
	}
}

// SetDebounceInterval configures the in-memory throttle window for heartbeat writes.
// Setting interval to 0 disables debouncing.
func (r *DeviceRepository) SetDebounceInterval(interval time.Duration) {
	r.debounceMu.Lock()
	defer r.debounceMu.Unlock()
	r.debounceInterval = interval
}

// SetDB updates the database connection and dialect at runtime.
func (r *DeviceRepository) SetDB(db *sql.DB, driver string) {
	r.mu.Lock()
	r.db = db
	r.driver = driver
	r.mu.Unlock()

	r.debounceMu.Lock()
	r.lastPersisted = make(map[string]time.Time)
	r.debounceMu.Unlock()
}

func (r *DeviceRepository) getDB() (*sql.DB, string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.db, r.driver
}

// GetAllDevices retrieves all registered systems for the dashboard.
// It supplements persisted data with hot in-memory timestamps from recent heartbeats.
func (r *DeviceRepository) GetAllDevices() ([]domain.Device, error) {
	db, _ := r.getDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `SELECT id, name, identifier, last_seen, enabled FROM devices ORDER BY name ASC;`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query devices: %w", err)
	}
	defer rows.Close()

	var devices []domain.Device
	for rows.Next() {
		var d domain.Device
		var lastSeen sql.NullTime

		if err := rows.Scan(&d.ID, &d.Name, &d.Identifier, &lastSeen, &d.Enabled); err != nil {
			return nil, err
		}

		if lastSeen.Valid {
			d.LastSeen = lastSeen.Time
		}
		devices = append(devices, d)
	}

	// Overlay warm in-memory heartbeat timestamps for real-time dashboard accuracy
	r.debounceMu.Lock()
	for i := range devices {
		if cached, ok := r.lastPersisted[devices[i].Identifier]; ok && cached.After(devices[i].LastSeen) {
			devices[i].LastSeen = cached
		}
	}
	r.debounceMu.Unlock()

	return devices, nil
}

// UpdateLastSeen updates the heartbeat for a specific system.
// AUTO-DISCOVERY: If the 'identifier' (origin) doesn't exist, it creates it.
// DEBOUNCED: If an update for the same device occurred within debounceInterval,
// the in-memory timestamp is updated immediately and database write is debounced.
func (r *DeviceRepository) UpdateLastSeen(identifier string, seenAt time.Time) error {
	if seenAt.IsZero() {
		seenAt = time.Now()
	}

	r.debounceMu.Lock()
	if r.lastPersisted == nil {
		r.lastPersisted = make(map[string]time.Time)
	}
	last, exists := r.lastPersisted[identifier]
	interval := r.debounceInterval
	// If recently persisted and within debounce window, update in-memory timestamp and skip DB disk write
	if interval > 0 && exists && seenAt.Sub(last) < interval && time.Since(last) < interval {
		r.lastPersisted[identifier] = seenAt
		r.debounceMu.Unlock()
		return nil
	}
	r.lastPersisted[identifier] = seenAt
	r.debounceMu.Unlock()

	return r.ForceUpdateLastSeen(identifier, seenAt)
}

// ForceUpdateLastSeen immediately persists the heartbeat to the database, bypassing the debounce cache.
func (r *DeviceRepository) ForceUpdateLastSeen(identifier string, seenAt time.Time) error {
	if seenAt.IsZero() {
		seenAt = time.Now()
	}

	db, driver := r.getDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	query := `
	INSERT INTO devices (name, identifier, last_seen, enabled)
	VALUES (?, ?, ?, 1)
	ON CONFLICT(identifier) DO UPDATE SET last_seen = excluded.last_seen, enabled = 1;
	`
	if driver == "postgres" {
		query = `
		INSERT INTO devices (name, identifier, last_seen, enabled)
		VALUES ($1, $2, $3, TRUE)
		ON CONFLICT(identifier) DO UPDATE SET last_seen = EXCLUDED.last_seen, enabled = TRUE;
		`
	}

	defaultName := identifier

	_, err := db.Exec(query, defaultName, identifier, seenAt)
	if err != nil {
		return fmt.Errorf("failed to update heartbeat for %s: %w", identifier, err)
	}

	return nil
}

// DeleteDevice removes a system permanently by its identifier and clears its debounce cache.
func (r *DeviceRepository) DeleteDevice(identifier string) error {
	r.debounceMu.Lock()
	if r.lastPersisted != nil {
		delete(r.lastPersisted, identifier)
	}
	r.debounceMu.Unlock()

	db, driver := r.getDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	query := AdaptQuery(`DELETE FROM devices WHERE identifier = ?;`, driver)
	_, err := db.Exec(query, identifier)
	return err
}
