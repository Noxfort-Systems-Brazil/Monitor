// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: internal/storage/telemetry_repo.go
// Author: Gabriel Moraes
// Date: 2026-01-19
// Modified: 2026-09-04 (PostgreSQL & Hot-Reload Support)

package storage

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"noxfort-monitor-server/internal/domain"
)

// TelemetryRecord pairs a device identifier with its incoming event for batch persistence.
type TelemetryRecord struct {
	Identifier string
	Event      *domain.IncomingEvent
}

// TelemetryRepository implements the storage logic for incidents and logs.
type TelemetryRepository struct {
	mu     sync.RWMutex
	db     *sql.DB
	driver string
}

// NewTelemetryRepository creates a new instance.
func NewTelemetryRepository(db *sql.DB) *TelemetryRepository {
	return &TelemetryRepository{db: db, driver: "sqlite"}
}

// SetDB updates the database connection and dialect at runtime.
func (r *TelemetryRepository) SetDB(db *sql.DB, driver string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.db = db
	r.driver = driver
}

func (r *TelemetryRepository) getDB() (*sql.DB, string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.db, r.driver
}

// SaveEvent persists an incident into the database.
func (r *TelemetryRepository) SaveEvent(identifier string, event *domain.IncomingEvent) error {
	db, driver := r.getDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	query := AdaptQuery(`
		INSERT INTO telemetry (category, device_id, origin, level, message, occurred_at)
		VALUES (?, ?, ?, ?, ?, ?);
	`, driver)

	_, err := db.Exec(query,
		event.Category,
		identifier,
		event.Origin,
		event.Level,
		event.Message,
		event.OccurredAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert telemetry event: %w", err)
	}

	return nil
}

// SaveEventsBatch inserts multiple telemetry events in a single SQL operation.
// This drastically minimizes network round-trips and PostgreSQL WAL fsync overhead.
func (r *TelemetryRepository) SaveEventsBatch(records []TelemetryRecord) error {
	if len(records) == 0 {
		return nil
	}

	db, driver := r.getDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	var queryBuilder strings.Builder
	queryBuilder.WriteString("INSERT INTO telemetry (category, device_id, origin, level, message, occurred_at) VALUES ")

	args := make([]interface{}, 0, len(records)*6)
	for i, rec := range records {
		if i > 0 {
			queryBuilder.WriteString(", ")
		}
		if driver == "postgres" {
			base := i * 6
			queryBuilder.WriteString(fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d)",
				base+1, base+2, base+3, base+4, base+5, base+6))
		} else {
			queryBuilder.WriteString("(?, ?, ?, ?, ?, ?)")
		}

		args = append(args,
			rec.Event.Category,
			rec.Identifier,
			rec.Event.Origin,
			rec.Event.Level,
			rec.Event.Message,
			rec.Event.OccurredAt,
		)
	}
	queryBuilder.WriteString(";")

	_, err := db.Exec(queryBuilder.String(), args...)
	if err != nil {
		return fmt.Errorf("failed to batch insert %d telemetry events: %w", len(records), err)
	}

	return nil
}

// GetRecentIncidents retrieves the last N events for display in the UI.
func (r *TelemetryRepository) GetRecentIncidents(limit int) ([]domain.IncomingEvent, error) {
	db, driver := r.getDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := AdaptQuery(`
		SELECT category, origin, level, message, occurred_at 
		FROM telemetry 
		ORDER BY occurred_at DESC 
		LIMIT ?;
	`, driver)

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent incidents: %w", err)
	}
	defer rows.Close()

	var events []domain.IncomingEvent

	for rows.Next() {
		var e domain.IncomingEvent
		var catStr, levelStr string

		if err := rows.Scan(&catStr, &e.Origin, &levelStr, &e.Message, &e.OccurredAt); err != nil {
			log.Printf("[STORAGE] Warning: failed to scan event row: %v", err)
			continue
		}

		e.Category = domain.EventCategory(catStr)
		e.Level = domain.EventLevel(levelStr)

		events = append(events, e)
	}

	return events, nil
}

// GetRecentIncidentsByDevice retrieves the last N events for a specific device.
// Utilizes the composite index idx_telemetry_device_occurred for instant index scan.
func (r *TelemetryRepository) GetRecentIncidentsByDevice(deviceID string, limit int) ([]domain.IncomingEvent, error) {
	db, driver := r.getDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := AdaptQuery(`
		SELECT category, origin, level, message, occurred_at 
		FROM telemetry 
		WHERE device_id = ?
		ORDER BY occurred_at DESC 
		LIMIT ?;
	`, driver)

	rows, err := db.Query(query, deviceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query incidents for device %s: %w", deviceID, err)
	}
	defer rows.Close()

	var events []domain.IncomingEvent
	for rows.Next() {
		var e domain.IncomingEvent
		var catStr, levelStr string

		if err := rows.Scan(&catStr, &e.Origin, &levelStr, &e.Message, &e.OccurredAt); err != nil {
			log.Printf("[STORAGE] Warning: failed to scan event row: %v", err)
			continue
		}

		e.Category = domain.EventCategory(catStr)
		e.Level = domain.EventLevel(levelStr)
		events = append(events, e)
	}

	return events, nil
}

// BufferedTelemetryWriter collects events in an in-memory queue and flushes them in batches.
type BufferedTelemetryWriter struct {
	repo          *TelemetryRepository
	queue         chan TelemetryRecord
	flushInterval time.Duration
	batchSize     int
	stopChan      chan struct{}
	doneChan      chan struct{}
	mu            sync.Mutex
	closed        bool
}

// NewBufferedTelemetryWriter starts a background batch ingestion worker.
func NewBufferedTelemetryWriter(repo *TelemetryRepository, batchSize int, flushInterval time.Duration) *BufferedTelemetryWriter {
	if batchSize <= 0 {
		batchSize = 50
	}
	if flushInterval <= 0 {
		flushInterval = 100 * time.Millisecond
	}

	w := &BufferedTelemetryWriter{
		repo:          repo,
		queue:         make(chan TelemetryRecord, batchSize*4),
		flushInterval: flushInterval,
		batchSize:     batchSize,
		stopChan:      make(chan struct{}),
		doneChan:      make(chan struct{}),
	}
	go w.worker()
	return w
}

// Enqueue adds a telemetry event to the buffered batch writer.
// If the buffer is full, it immediately falls back to synchronous write to avoid dropping data.
func (w *BufferedTelemetryWriter) Enqueue(identifier string, event *domain.IncomingEvent) bool {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return false
	}
	w.mu.Unlock()

	select {
	case w.queue <- TelemetryRecord{Identifier: identifier, Event: event}:
		return true
	default:
		// Queue full fallback
		_ = w.repo.SaveEvent(identifier, event)
		return true
	}
}

func (w *BufferedTelemetryWriter) worker() {
	defer close(w.doneChan)
	ticker := time.NewTicker(w.flushInterval)
	defer ticker.Stop()

	batch := make([]TelemetryRecord, 0, w.batchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		if err := w.repo.SaveEventsBatch(batch); err != nil {
			log.Printf("[STORAGE] Buffered write error: %v", err)
		}
		batch = batch[:0]
	}

	for {
		select {
		case <-w.stopChan:
			for {
				select {
				case item := <-w.queue:
					batch = append(batch, item)
					if len(batch) >= w.batchSize {
						flush()
					}
				default:
					flush()
					return
				}
			}
		case item := <-w.queue:
			batch = append(batch, item)
			if len(batch) >= w.batchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

// Close gracefully flushes all queued items and terminates the worker.
func (w *BufferedTelemetryWriter) Close() {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return
	}
	w.closed = true
	w.mu.Unlock()

	close(w.stopChan)
	<-w.doneChan
}
