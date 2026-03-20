// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.
//
// File: internal/storage/telemetry_repo.go
// Author: Gabriel Moraes
// Date: 2026-01-19

package storage

import (
	"database/sql"
	"fmt"
	"log"

	"noxfort-monitor-server/internal/domain"
)

// TelemetryRepository implements the storage logic for incidents and logs.
type TelemetryRepository struct {
	db *sql.DB
}

// NewTelemetryRepository creates a new instance connected to the SQLite DB.
func NewTelemetryRepository(db *sql.DB) *TelemetryRepository {
	return &TelemetryRepository{db: db}
}

// SaveEvent persists an incident into the database.
func (r *TelemetryRepository) SaveEvent(identifier string, event *domain.IncomingEvent) error {
	query := `
		INSERT INTO telemetry (category, device_id, origin, level, message, occurred_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	// We use the "identifier" as device_id and "origin" as origin.
	// Usually they are the same, but identifier is the internal reference.
	_, err := r.db.Exec(query,
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

// GetRecentIncidents retrieves the last N events for display in the UI.
func (r *TelemetryRepository) GetRecentIncidents(limit int) ([]domain.IncomingEvent, error) {
	query := `
		SELECT category, origin, level, message, occurred_at 
		FROM telemetry 
		ORDER BY occurred_at DESC 
		LIMIT ?
	`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent incidents: %w", err)
	}
	defer rows.Close()

	var events []domain.IncomingEvent

	for rows.Next() {
		var e domain.IncomingEvent
		// Helper variables for scanning (SQLite stores strings)
		var catStr, levelStr string

		if err := rows.Scan(&catStr, &e.Origin, &levelStr, &e.Message, &e.OccurredAt); err != nil {
			log.Printf("[STORAGE] Warning: failed to scan event row: %v", err)
			continue
		}

		// Convert strings back to Domain Types
		e.Category = domain.EventCategory(catStr)
		e.Level = domain.EventLevel(levelStr)

		events = append(events, e)
	}

	return events, nil
}
