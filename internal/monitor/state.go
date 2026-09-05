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
// Modified: 2026-09-04 (SOLID Refactor)

package monitor

import (
	"log"
	"strings"
	"time"

	"noxfort-monitor-server/internal/domain"
)

// EventProcessor is the contract for components that ingest and process system telemetry messages.
type EventProcessor interface {
	ProcessEvent(identifier string, event *domain.IncomingEvent)
}

// DeviceHeartbeatUpdater records the latest activity timestamp of a device.
type DeviceHeartbeatUpdater interface {
	UpdateLastSeen(identifier string, lastSeen time.Time) error
}

// HeartbeatDetector determines whether an incoming event is a routine keep-alive or an incident.
type HeartbeatDetector interface {
	IsHeartbeat(event *domain.IncomingEvent) bool
}

// KeywordHeartbeatDetector detects heartbeats using INFO level and matching phrases.
type KeywordHeartbeatDetector struct {
	keywords []string
}

// NewKeywordHeartbeatDetector initializes a detector with default or custom keep-alive keywords.
func NewKeywordHeartbeatDetector(keywords ...string) *KeywordHeartbeatDetector {
	if len(keywords) == 0 {
		keywords = []string{"system ok", "heartbeat", "online", "ativo", "ok"}
	}
	return &KeywordHeartbeatDetector{keywords: keywords}
}

// IsHeartbeat returns true if the event has LevelInfo and contains keep-alive text.
func (d *KeywordHeartbeatDetector) IsHeartbeat(event *domain.IncomingEvent) bool {
	if event == nil || event.Level != domain.LevelInfo {
		return false
	}
	lower := strings.ToLower(event.Message)
	for _, kw := range d.keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// StateManager orchestrates incoming events, device heartbeats, incident audit logging, and alert triggers.
type StateManager struct {
	events   EventRecorder
	devices  DeviceHeartbeatUpdater
	alerts   AlertDispatcher
	detector HeartbeatDetector
}

// NewStateManager creates a new StateManager with default keyword heartbeat detection.
func NewStateManager(
	tRepo EventRecorder,
	dRepo DeviceHeartbeatUpdater,
	alerts AlertDispatcher,
) *StateManager {
	return NewStateManagerWithDetector(tRepo, dRepo, alerts, NewKeywordHeartbeatDetector())
}

// NewStateManagerWithDetector creates a StateManager with an injected HeartbeatDetector (OCP / DIP).
func NewStateManagerWithDetector(
	tRepo EventRecorder,
	dRepo DeviceHeartbeatUpdater,
	alerts AlertDispatcher,
	detector HeartbeatDetector,
) *StateManager {
	if detector == nil {
		detector = NewKeywordHeartbeatDetector()
	}
	return &StateManager{
		events:   tRepo,
		devices:  dRepo,
		alerts:   alerts,
		detector: detector,
	}
}

// ProcessEvent is the main entry point for incoming telemetry messages (MQTT / HTTP).
// It updates the system heartbeat, filters out keep-alives, persists incidents, and alerts operators.
func (sm *StateManager) ProcessEvent(identifier string, event *domain.IncomingEvent) {
	if event == nil {
		return
	}

	// 1. Heartbeat: Always update the system's last seen status.
	if sm.devices != nil {
		if err := sm.devices.UpdateLastSeen(identifier, event.OccurredAt); err != nil {
			log.Printf("[STATE] Failed to update heartbeat for %s: %v", identifier, err)
		}
	}

	// 2. Filter: Determine if this is a keep-alive heartbeat or an actual incident.
	if sm.detector.IsHeartbeat(event) {
		return
	}

	// 3. Incident Processing
	log.Printf("🚨 [INCIDENT] System: %s | Level: %s | Msg: %s", identifier, event.Level, event.Message)

	// A. Persist the Incident for Audit
	if sm.events != nil {
		if err := sm.events.SaveEvent(identifier, event); err != nil {
			log.Printf("[STATE] CRITICAL: Failed to save incident to DB: %v", err)
		}
	}

	// B. Trigger Alert Dispatch
	if sm.alerts != nil {
		sm.alerts.TriggerAlert(identifier, event)
	}
}
