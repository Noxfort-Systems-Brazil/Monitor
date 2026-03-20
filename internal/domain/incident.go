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
// File: internal/domain/incident.go
// Author: Gabriel Moraes
// Date: 2026-01-16

package domain

import (
	"time"
)

// IncidentStatus defines the lifecycle state of an alert.
type IncidentStatus string

const (
	IncidentStatusOpen     IncidentStatus = "OPEN"
	IncidentStatusResolved IncidentStatus = "RESOLVED"
)

// Incident represents a critical event detected by the Monitor.
type Incident struct {
	ID       int64  `json:"id"`
	DeviceID uint16 `json:"device_id"`

	// ErrorCode identifies the type of failure.
	// Updated to uint16 to support codes > 255 (e.g., 999 for Timeout).
	ErrorCode uint16 `json:"error_code"`

	StartTime time.Time      `json:"start_time"`
	EndTime   *time.Time     `json:"end_time,omitempty"` // Nullable if still open
	Status    IncidentStatus `json:"status"`
}
