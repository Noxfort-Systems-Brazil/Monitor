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
// File: internal/tunnel/manager_test.go
// Author: Gabriel Moraes
// Date: 2026-09-04
// Modified: 2026-09-04 (SOLID: MockDriver isolation testing)

package tunnel

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// mockDriver implements Driver for testing Manager state transitions in complete isolation.
type mockDriver struct {
	mu           sync.Mutex
	name         string
	available    bool
	started      bool
	startErr     error
	publicURL    string
	urlErr       error
	waitErr      error
	waitCh       chan struct{}
}

func newMockDriver(available bool) *mockDriver {
	return &mockDriver{
		name:      "mock",
		available: available,
		waitCh:    make(chan struct{}),
	}
}

func (m *mockDriver) Name() string { return m.name }

func (m *mockDriver) IsAvailable() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.available
}

func (m *mockDriver) Start(ctx context.Context, cfg Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.startErr != nil {
		return m.startErr
	}
	m.started = true
	return nil
}

func (m *mockDriver) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = false
	return nil
}

func (m *mockDriver) GetPublicURL(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.urlErr != nil {
		return "", m.urlErr
	}
	return m.publicURL, nil
}

func (m *mockDriver) Wait() error {
	<-m.waitCh
	return m.waitErr
}

func TestManager_DefaultInitialization(t *testing.T) {
	mock := newMockDriver(true)
	mgr := NewManager(mock, "")

	if mgr.localPort != "8080" {
		t.Errorf("Expected default localPort 8080, got %s", mgr.localPort)
	}

	status := mgr.GetStatus()
	if status.State != StateOffline {
		t.Errorf("Expected initial state OFFLINE, got %s", status.State)
	}
	if !mgr.IsBinaryAvailable() {
		t.Errorf("Expected binary to be available according to mock")
	}
}

func TestManager_UnavailableDriver(t *testing.T) {
	mock := newMockDriver(false)
	mgr := NewManager(mock, "8080")

	err := mgr.Start("test_token", "test.domain")
	if err == nil {
		t.Errorf("Expected error when driver is unavailable, got nil")
	}

	status := mgr.GetStatus()
	if status.State != StateError {
		t.Errorf("Expected state ERROR, got %s", status.State)
	}
}

func TestManager_EmptyTokenValidation(t *testing.T) {
	mock := newMockDriver(true)
	mgr := NewManager(mock, "8080")

	err := mgr.Start("", "test.domain")
	if err == nil {
		t.Errorf("Expected error when token is empty, got nil")
	}

	status := mgr.GetStatus()
	if status.State != StateError {
		t.Errorf("Expected state ERROR, got %s", status.State)
	}
}

func TestManager_SuccessfulLifecycle(t *testing.T) {
	mock := newMockDriver(true)
	mock.publicURL = "https://noxfort.ngrok-free.app"

	mgr := NewManager(mock, "8080")
	err := mgr.Start("valid_token", "noxfort.ngrok-free.app")
	if err != nil {
		t.Fatalf("Unexpected error starting manager: %v", err)
	}

	// Verify stop
	err = mgr.Stop()
	if err != nil {
		t.Fatalf("Unexpected error stopping manager: %v", err)
	}

	status := mgr.GetStatus()
	if status.State != StateOffline {
		t.Errorf("Expected state OFFLINE after Stop, got %s", status.State)
	}
}

func TestManager_DriverStartFailure(t *testing.T) {
	mock := newMockDriver(true)
	mock.startErr = errors.New("simulated process start error")

	mgr := NewManager(mock, "8080")
	err := mgr.Start("valid_token", "domain")
	if err == nil {
		t.Fatalf("Expected error when driver fails to start, got nil")
	}

	status := mgr.GetStatus()
	if status.State != StateError {
		t.Errorf("Expected state ERROR, got %s", status.State)
	}
}
