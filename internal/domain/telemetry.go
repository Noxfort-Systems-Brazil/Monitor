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
// File: internal/domain/telemetry.go
// Author: Gabriel Moraes
// Date: 2026-01-19

package domain

import "time"

// EventLevel defines the severity of the incident.
type EventLevel string

const (
	LevelInfo     EventLevel = "INFO"
	LevelWarning  EventLevel = "WARNING"
	LevelCritical EventLevel = "CRITICAL"
)

// EventCategory defines the nature of the problem (Who resolves it?).
type EventCategory string

const (
	CategoryHardware EventCategory = "HARDWARE" // Dispatched to Technician
	CategorySoftware EventCategory = "SOFTWARE" // Dispatched to Programmer
)

// IncomingEvent represents the Universal JSON payload received from devices.
// Structure: Category -> Origin -> Level -> Message -> Time
type IncomingEvent struct {
	// Category defines if it's a HARDWARE or SOFTWARE issue.
	// This directs the alert to the correct specialist.
	Category EventCategory `json:"category"`

	// Origin is the system identifier (e.g. "synapse", "carina").
	Origin string `json:"origin"`

	// Level is the severity (INFO, WARNING, CRITICAL).
	Level EventLevel `json:"level"`

	// Message is the human-readable description of the event.
	Message string `json:"message"`

	// OccurredAt is the exact timestamp from the device (Source of Truth).
	OccurredAt time.Time `json:"occurred_at"`
}

// TelemetryRepository defines the contract for persisting and retrieving logs.
type TelemetryRepository interface {
	// SaveEvent persists a new incident to the immutable log.
	SaveEvent(identifier string, event *IncomingEvent) error

	// GetRecentIncidents retrieves the last N events for the Dashboard display.
	GetRecentIncidents(limit int) ([]IncomingEvent, error)
}
