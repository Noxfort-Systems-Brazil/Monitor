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
// File: internal/monitor/engine_test.go
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

type mockDeviceLister struct {
	devices []domain.Device
	err     error
}

func (m *mockDeviceLister) GetAllDevices() ([]domain.Device, error) {
	return m.devices, m.err
}

type mockEventRecorder struct {
	mu     sync.Mutex
	err    error
	events []struct {
		identifier string
		event      *domain.IncomingEvent
	}
}

func (m *mockEventRecorder) SaveEvent(identifier string, event *domain.IncomingEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, struct {
		identifier string
		event      *domain.IncomingEvent
	}{identifier: identifier, event: event})
	return m.err
}

type mockAlertDispatcher struct {
	mu     sync.Mutex
	alerts []struct {
		identifier string
		event      *domain.IncomingEvent
	}
}

func (m *mockAlertDispatcher) TriggerAlert(identifier string, event *domain.IncomingEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alerts = append(m.alerts, struct {
		identifier string
		event      *domain.IncomingEvent
	}{identifier: identifier, event: event})
}

func TestEngine_Stop_Idempotent(t *testing.T) {
	engine := NewEngine(nil, nil, nil)
	engine.Start()

	time.Sleep(50 * time.Millisecond)

	// First stop
	done := make(chan struct{})
	go func() {
		engine.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Engine.Stop() deadlocked on first call")
	}

	// Second stop (must not deadlock)
	done2 := make(chan struct{})
	go func() {
		engine.Stop()
		close(done2)
	}()

	select {
	case <-done2:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Engine.Stop() deadlocked on second call")
	}

	// Third stop (must not deadlock)
	engine.Stop()
}

func TestEngine_OfflineDetectionAntiSpamAndRecovery(t *testing.T) {
	now := time.Now()

	devActive := domain.Device{
		Identifier: "DEV-ONLINE",
		Enabled:    true,
		LastSeen:   now.Add(-1 * time.Minute), // 1 min ago (threshold is 5 min)
	}

	devSilent := domain.Device{
		Identifier: "DEV-SILENT",
		Enabled:    true,
		LastSeen:   now.Add(-10 * time.Minute), // 10 min ago (> 5 min threshold)
	}

	devDisabled := domain.Device{
		Identifier: "DEV-DISABLED",
		Enabled:    false,
		LastSeen:   now.Add(-60 * time.Minute), // disabled
	}

	devices := &mockDeviceLister{
		devices: []domain.Device{devActive, devSilent, devDisabled},
	}
	events := &mockEventRecorder{}
	alerts := &mockAlertDispatcher{}

	cfg := EngineConfig{
		CheckInterval:    10 * time.Millisecond,
		OfflineThreshold: 5 * time.Minute,
	}

	engine := NewEngineWithConfig(devices, events, alerts, cfg)

	// 1. First run: only DEV-SILENT should trigger an OFFLINE alert
	engine.CheckSystemsOnce()

	alerts.mu.Lock()
	if len(alerts.alerts) != 1 {
		t.Fatalf("Expected exactly 1 alert on first check, got %d", len(alerts.alerts))
	}
	if alerts.alerts[0].identifier != "DEV-SILENT" || alerts.alerts[0].event.Level != domain.LevelCritical {
		t.Errorf("Expected critical offline alert for DEV-SILENT, got: %+v", alerts.alerts[0])
	}
	alerts.mu.Unlock()

	// 2. Second run with same state: anti-spam MUST prevent repeated alert
	engine.CheckSystemsOnce()

	alerts.mu.Lock()
	if len(alerts.alerts) != 1 {
		t.Errorf("Anti-spam failed: expected still 1 alert, got %d", len(alerts.alerts))
	}
	alerts.mu.Unlock()

	// 3. Recovery: DEV-SILENT comes back online (LastSeen updated to now)
	devices.devices[1].LastSeen = time.Now()
	engine.CheckSystemsOnce()

	alerts.mu.Lock()
	if len(alerts.alerts) != 2 {
		t.Fatalf("Expected recovery alert to be dispatched (total 2), got %d", len(alerts.alerts))
	}
	recoveryAlert := alerts.alerts[1]
	if recoveryAlert.identifier != "DEV-SILENT" || recoveryAlert.event.Level != domain.LevelInfo {
		t.Errorf("Expected info recovery alert for DEV-SILENT, got: %+v", recoveryAlert)
	}
	alerts.mu.Unlock()

	// 4. Following run: system is now stably online, no more alerts
	engine.CheckSystemsOnce()

	alerts.mu.Lock()
	if len(alerts.alerts) != 2 {
		t.Errorf("Expected no new alerts after stable recovery, got %d", len(alerts.alerts))
	}
	alerts.mu.Unlock()
}

func TestEngine_DeviceListerErrorsAndNil(t *testing.T) {
	// 1. Nil devices lister
	engineNil := NewEngine(nil, nil, nil)
	engineNil.CheckSystemsOnce() // must not panic

	// 2. Devices query error
	mockErr := &mockDeviceLister{err: errors.New("db disk I/O error")}
	mockAlerts := &mockAlertDispatcher{}
	mockEvents := &mockEventRecorder{err: errors.New("cannot save")}
	engineErr := NewEngine(mockErr, mockEvents, mockAlerts)
	engineErr.CheckSystemsOnce() // must log and return gracefully

	if len(mockAlerts.alerts) != 0 {
		t.Errorf("Expected no alerts when device query fails")
	}

	// 3. Synthetic event with failing event recorder (must still trigger alert)
	engineErr.triggerEvent("SYS-FAIL", domain.LevelCritical, "Test message")
	if len(mockAlerts.alerts) != 1 {
		t.Errorf("Expected alert to be dispatched even if event saving fails")
	}
}

func TestEngine_ConfigDefaults(t *testing.T) {
	// Zero values should trigger fallback defaults
	cfg := EngineConfig{CheckInterval: 0, OfflineThreshold: 0}
	engine := NewEngineWithConfig(nil, nil, nil, cfg)

	if engine.config.CheckInterval <= 0 {
		t.Errorf("Expected positive default CheckInterval, got %v", engine.config.CheckInterval)
	}
	if engine.config.OfflineThreshold <= 0 {
		t.Errorf("Expected positive default OfflineThreshold, got %v", engine.config.OfflineThreshold)
	}
}
