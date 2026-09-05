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
// File: internal/monitor/state_test.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package monitor

import (
	"errors"
	"sync"
	"testing"
	"time"

	"noxfort-monitor-server/internal/domain"
)

type mockHeartbeatUpdater struct {
	mu         sync.Mutex
	calls      int
	lastSeen   time.Time
	identifier string
	err        error
}

func (m *mockHeartbeatUpdater) UpdateLastSeen(identifier string, lastSeen time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	m.identifier = identifier
	m.lastSeen = lastSeen
	return m.err
}

func TestKeywordHeartbeatDetector(t *testing.T) {
	detector := NewKeywordHeartbeatDetector()

	// 1. Info level with standard keywords
	if !detector.IsHeartbeat(&domain.IncomingEvent{Level: domain.LevelInfo, Message: "System OK"}) {
		t.Errorf("Expected 'System OK' at INFO level to be detected as heartbeat")
	}
	if !detector.IsHeartbeat(&domain.IncomingEvent{Level: domain.LevelInfo, Message: "Periodic heartbeat ping"}) {
		t.Errorf("Expected 'heartbeat' at INFO level to be detected as heartbeat")
	}
	if !detector.IsHeartbeat(&domain.IncomingEvent{Level: domain.LevelInfo, Message: "Sensor ONLINE"}) {
		t.Errorf("Expected 'online' at INFO level to be detected as heartbeat")
	}
	if !detector.IsHeartbeat(&domain.IncomingEvent{Level: domain.LevelInfo, Message: "Dispositivo ATIVO"}) {
		t.Errorf("Expected 'ativo' at INFO level to be detected as heartbeat")
	}

	// 2. Non-info level must NEVER be filtered as heartbeat
	if detector.IsHeartbeat(&domain.IncomingEvent{Level: domain.LevelWarning, Message: "System OK but temp high"}) {
		t.Errorf("WARNING events must not be filtered as heartbeat")
	}
	if detector.IsHeartbeat(&domain.IncomingEvent{Level: domain.LevelCritical, Message: "Heartbeat failed: power failure"}) {
		t.Errorf("CRITICAL events must not be filtered as heartbeat")
	}

	// 3. Custom keywords
	custom := NewKeywordHeartbeatDetector("pong", "alive")
	if !custom.IsHeartbeat(&domain.IncomingEvent{Level: domain.LevelInfo, Message: "PONG"}) {
		t.Errorf("Custom keyword 'PONG' should be recognized")
	}
	if custom.IsHeartbeat(&domain.IncomingEvent{Level: domain.LevelInfo, Message: "System OK"}) {
		t.Errorf("Standard keywords should not match when overridden with custom set")
	}
}

func TestStateManager_HeartbeatVsIncident(t *testing.T) {
	devices := &mockHeartbeatUpdater{}
	events := &mockEventRecorder{}
	alerts := &mockAlertDispatcher{}

	sm := NewStateManager(events, devices, alerts)
	now := time.Now()

	// 1. Send Heartbeat event
	heartbeat := &domain.IncomingEvent{
		Origin:     "PLC-01",
		Level:      domain.LevelInfo,
		Message:    "System OK: normal operations",
		OccurredAt: now,
	}

	sm.ProcessEvent("PLC-01", heartbeat)

	if devices.calls != 1 || devices.identifier != "PLC-01" {
		t.Errorf("Expected 1 heartbeat update for PLC-01, got %d", devices.calls)
	}
	if len(events.events) != 0 {
		t.Errorf("Heartbeats should NOT be saved as incidents in DB")
	}
	if len(alerts.alerts) != 0 {
		t.Errorf("Heartbeats should NOT trigger alert notifications")
	}

	// 2. Send Incident event
	incident := &domain.IncomingEvent{
		Origin:     "PLC-01",
		Level:      domain.LevelCritical,
		Message:    "Emergency Stop Triggered",
		OccurredAt: now,
	}

	sm.ProcessEvent("PLC-01", incident)

	if devices.calls != 2 {
		t.Errorf("Expected heartbeat to be updated even during incidents")
	}
	if len(events.events) != 1 {
		t.Fatalf("Expected incident to be saved to DB, got %d", len(events.events))
	}
	if events.events[0].identifier != "PLC-01" || events.events[0].event.Message != "Emergency Stop Triggered" {
		t.Errorf("Saved event does not match incident: %+v", events.events[0])
	}
	if len(alerts.alerts) != 1 {
		t.Fatalf("Expected incident to trigger alert dispatch, got %d", len(alerts.alerts))
	}
	if alerts.alerts[0].identifier != "PLC-01" {
		t.Errorf("Alert triggered for wrong system: %s", alerts.alerts[0].identifier)
	}
}

func TestStateManager_HeartbeatFailureDoesNotBlockIncident(t *testing.T) {
	devices := &mockHeartbeatUpdater{err: errors.New("db locked")}
	events := &mockEventRecorder{}
	alerts := &mockAlertDispatcher{}

	sm := NewStateManager(events, devices, alerts)

	incident := &domain.IncomingEvent{
		Origin:     "CONVEYOR-02",
		Level:      domain.LevelWarning,
		Message:    "Motor Current High (42A)",
		OccurredAt: time.Now(),
	}

	sm.ProcessEvent("CONVEYOR-02", incident)

	if len(events.events) != 1 {
		t.Errorf("Incident must be saved even if heartbeat update fails")
	}
	if len(alerts.alerts) != 1 {
		t.Errorf("Alert must be dispatched even if heartbeat update fails")
	}
}
