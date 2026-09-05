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
// File: internal/transport/http/tunnel_handler_test.go
// Author: Gabriel Moraes
// Date: 2026-09-04
// Modified: 2026-09-04 (SOLID: MockTunnelService isolation)

package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"noxfort-monitor-server/internal/domain"
	"noxfort-monitor-server/internal/tunnel"
)

type mockTunnelService struct {
	mu           sync.Mutex
	status       tunnel.Status
	startErr     error
	stopErr      error
	startedToken string
	startedDom   string
	isAvailable  bool
}

func (m *mockTunnelService) Start(authToken, domain string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startedToken = authToken
	m.startedDom = domain
	if m.startErr != nil {
		return m.startErr
	}
	m.status.State = tunnel.StateOnline
	m.status.PublicURL = "https://" + domain
	m.status.TelemetryURL = "https://" + domain + "/api/telemetry"
	return nil
}

func (m *mockTunnelService) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stopErr != nil {
		return m.stopErr
	}
	m.status.State = tunnel.StateOffline
	m.status.PublicURL = ""
	m.status.TelemetryURL = ""
	return nil
}

func (m *mockTunnelService) GetStatus() tunnel.Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status
}

func (m *mockTunnelService) IsBinaryAvailable() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.isAvailable
}

func TestTunnelHandler_HandleStatus(t *testing.T) {
	repo := &mockSettingsRepoForHandler{
		settings: &domain.Settings{
			NgrokAuthToken: "tok_123",
			NgrokDomain:    "test.ngrok-free.app",
		},
	}
	mockSvc := &mockTunnelService{
		status: tunnel.Status{State: tunnel.StateOffline},
	}
	handler := NewTunnelHandler(repo, mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/tunnel/status", nil)
	rec := httptest.NewRecorder()

	handler.HandleStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rec.Code)
	}

	var status tunnel.Status
	if err := json.NewDecoder(rec.Body).Decode(&status); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if status.State != tunnel.StateOffline {
		t.Errorf("Expected initial state OFFLINE, got %s", status.State)
	}
}

func TestTunnelHandler_HandleSave(t *testing.T) {
	repo := &mockSettingsRepoForHandler{
		settings: &domain.Settings{},
	}
	mockSvc := &mockTunnelService{}
	handler := NewTunnelHandler(repo, mockSvc)

	formData := url.Values{}
	formData.Set("ngrok_auth_token", "my_secret_token_123")
	formData.Set("ngrok_domain", "custom.ngrok-free.app")
	formData.Set("ngrok_enabled", "on")

	req := httptest.NewRequest(http.MethodPost, "/api/tunnel/save", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.HandleSave(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("Expected redirect status 303, got %d", rec.Code)
	}

	saved, _ := repo.GetSettings()
	if saved.NgrokAuthToken != "my_secret_token_123" {
		t.Errorf("Expected saved token 'my_secret_token_123', got '%s'", saved.NgrokAuthToken)
	}
	if saved.NgrokDomain != "custom.ngrok-free.app" {
		t.Errorf("Expected saved domain 'custom.ngrok-free.app', got '%s'", saved.NgrokDomain)
	}
	if !saved.NgrokEnabled {
		t.Errorf("Expected NgrokEnabled to be true, got false")
	}

	if mockSvc.startedToken != "my_secret_token_123" {
		t.Errorf("Expected mock service to have started with token 'my_secret_token_123', got '%s'", mockSvc.startedToken)
	}
}

func TestTunnelHandler_HandleStartWithoutToken(t *testing.T) {
	repo := &mockSettingsRepoForHandler{
		settings: &domain.Settings{},
	}
	mockSvc := &mockTunnelService{}
	handler := NewTunnelHandler(repo, mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/tunnel/start", nil)
	rec := httptest.NewRecorder()

	handler.HandleStart(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400 when starting without token, got %d", rec.Code)
	}
}

func TestTunnelHandler_HandleStartFailure(t *testing.T) {
	repo := &mockSettingsRepoForHandler{
		settings: &domain.Settings{
			NgrokAuthToken: "token",
			NgrokDomain:    "domain",
		},
	}
	mockSvc := &mockTunnelService{
		startErr: errors.New("simulated error"),
	}
	handler := NewTunnelHandler(repo, mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/tunnel/start", nil)
	rec := httptest.NewRecorder()

	handler.HandleStart(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("Expected status 500 when start fails, got %d", rec.Code)
	}
}

func TestTunnelHandler_HandleStop(t *testing.T) {
	repo := &mockSettingsRepoForHandler{settings: &domain.Settings{}}
	mockSvc := &mockTunnelService{}
	handler := NewTunnelHandler(repo, mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/tunnel/stop", nil)
	rec := httptest.NewRecorder()

	handler.HandleStop(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rec.Code)
	}
}
