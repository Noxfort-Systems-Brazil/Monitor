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
// File: internal/domain/device.go
// Author: Gabriel Moraes
// Date: 2026-01-19

package domain

import "time"

// Device represents a monitored system/service (e.g. "synapse", "carina").
// It maps the 'origin' from the incoming JSON to a tracked state.
type Device struct {
	// ID is the internal database identifier.
	// Required for CRUD operations in the UI.
	ID int64 `json:"id"`

	// Name is a friendly label for the UI (e.g. "Synapse Service").
	Name string `json:"name"`

	// Identifier is the unique key (matches the "origin" field in JSON).
	// Replaces the old "HardwareID".
	Identifier string `json:"identifier"`

	// LastSeen tracks the last time we received ANY message (Heartbeat or Alert).
	LastSeen time.Time `json:"last_seen"`

	// Enabled allows ignoring a system during maintenance.
	Enabled bool `json:"enabled"`
}

// DeviceRepository defines the contract for persisting system states.
type DeviceRepository interface {
	// GetAllDevices returns all monitored systems.
	GetAllDevices() ([]Device, error)

	// UpdateLastSeen updates the heartbeat using the identifier (origin).
	UpdateLastSeen(identifier string, seenAt time.Time) error

	// DeleteDevice removes a system from monitoring.
	// We use the identifier (string) as the key for deletion.
	DeleteDevice(identifier string) error
}
