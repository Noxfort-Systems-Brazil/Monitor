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
// File: internal/monitor/state.go
// Author: Gabriel Moraes
// Date: 2026-01-19

package monitor

import (
	"log"
	"strings"

	"noxfort-monitor-server/internal/domain"
)

// StateManager orchestrates the logic between incoming events, storage, and alerts.
// It acts as the central decision maker of the system.
type StateManager struct {
	telemetryRepo domain.TelemetryRepository
	deviceRepo    domain.DeviceRepository
	alertService  *AlertService
}

// NewStateManager creates a new instance with all required dependencies injected.
func NewStateManager(
	tRepo domain.TelemetryRepository,
	dRepo domain.DeviceRepository,
	alerts *AlertService,
) *StateManager {
	return &StateManager{
		telemetryRepo: tRepo,
		deviceRepo:    dRepo,
		alertService:  alerts,
	}
}

// ProcessEvent is the main entry point for incoming messages (MQTT/HTTP).
// It applies the "Filter & Act" logic using the Universal Identifier.
func (sm *StateManager) ProcessEvent(identifier string, event *domain.IncomingEvent) {
	// 1. Heartbeat: Always update the system's last seen status.
	// This ensures the Watchdog knows the system is alive, even if it's reporting an error.
	if err := sm.deviceRepo.UpdateLastSeen(identifier, event.OccurredAt); err != nil {
		log.Printf("[STATE] Failed to update heartbeat for %s: %v", identifier, err)
		// We continue, as failing to update heartbeat shouldn't block alert processing.
	}

	// 2. Filter: Check if this is a "Noise" message (System OK) or an "Incident".
	// Using the Universal JSON "level" field to decide.
	isHeartbeat := event.Level == domain.LevelInfo && isSystemOkMessage(event.Message)

	if isHeartbeat {
		// It's just a heartbeat. We already updated LastSeen.
		// No need to save to DB or Alert.
		return
	}

	// 3. Incident Processing
	log.Printf("🚨 [INCIDENT] System: %s | Level: %s | Msg: %s", identifier, event.Level, event.Message)

	// A. Persist the Incident for Audit
	if err := sm.telemetryRepo.SaveEvent(identifier, event); err != nil {
		log.Printf("[STATE] CRITICAL: Failed to save incident to DB: %v", err)
	}

	// B. Trigger Human Notification (SMS/Email)
	// We pass the raw data to the AlertService which handles formatting and contacts.
	sm.alertService.TriggerAlert(identifier, event)
}

// isSystemOkMessage checks if the message is a standard keep-alive message.
// This allows us to filter out "System OK" or "Heartbeat" texts.
func isSystemOkMessage(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "system ok") ||
		strings.Contains(lower, "heartbeat") ||
		strings.Contains(lower, "online")
}
