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
// File: internal/storage/device_repo.go
// Author: Gabriel Moraes
// Date: 2026-01-19

package storage

import (
	"database/sql"
	"fmt"
	"time"

	"noxfort-monitor-server/internal/domain"
)

// DeviceRepository manages the lifecycle of monitored systems in the database.
type DeviceRepository struct {
	db *sql.DB
}

// NewDeviceRepository initializes the repository.
func NewDeviceRepository(db *sql.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

// GetAllDevices retrieves all registered systems for the dashboard.
func (r *DeviceRepository) GetAllDevices() ([]domain.Device, error) {
	// Updated query to match the simplified schema (identifier instead of hardware_id)
	query := `SELECT id, name, identifier, last_seen, enabled FROM devices ORDER BY name ASC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query devices: %w", err)
	}
	defer rows.Close()

	var devices []domain.Device
	for rows.Next() {
		var d domain.Device
		var lastSeen sql.NullTime

		// Scanning only the fields defined in the new Domain
		if err := rows.Scan(&d.ID, &d.Name, &d.Identifier, &lastSeen, &d.Enabled); err != nil {
			return nil, err
		}

		if lastSeen.Valid {
			d.LastSeen = lastSeen.Time
		}
		devices = append(devices, d)
	}
	return devices, nil
}

// UpdateLastSeen updates the heartbeat for a specific system.
// AUTO-DISCOVERY: If the 'identifier' (origin) doesn't exist, it creates it.
func (r *DeviceRepository) UpdateLastSeen(identifier string, seenAt time.Time) error {
	// UPSERT FIX:
	// We added ", enabled = 1" to the UPDATE clause.
	// This ensures that any device sending a heartbeat is automatically marked as ENABLED.
	query := `
	INSERT INTO devices (name, identifier, last_seen, enabled)
	VALUES (?, ?, ?, 1)
	ON CONFLICT(identifier) DO UPDATE SET last_seen = excluded.last_seen, enabled = 1;
	`

	// Default name is the Identifier itself (e.g. "synapse") until renamed
	defaultName := identifier

	_, err := r.db.Exec(query, defaultName, identifier, seenAt)
	if err != nil {
		return fmt.Errorf("failed to update heartbeat for %s: %w", identifier, err)
	}

	return nil
}

// DeleteDevice removes a system permanently by its identifier.
func (r *DeviceRepository) DeleteDevice(identifier string) error {
	query := `DELETE FROM devices WHERE identifier = ?`
	_, err := r.db.Exec(query, identifier)
	return err
}
