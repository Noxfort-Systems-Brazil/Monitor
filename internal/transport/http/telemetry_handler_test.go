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
// File: internal/transport/http/telemetry_handler_test.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"noxfort-monitor-server/internal/domain"
)

type mockTelemetryProcessor struct {
	mu     sync.Mutex
	events []struct {
		identifier string
		event      *domain.IncomingEvent
	}
}

func (m *mockTelemetryProcessor) ProcessEvent(identifier string, event *domain.IncomingEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, struct {
		identifier string
		event      *domain.IncomingEvent
	}{identifier: identifier, event: event})
}

func TestTelemetryHandler_HandleIngest(t *testing.T) {
	processor := &mockTelemetryProcessor{}
	handler := NewTelemetryHandler(processor)

	// 1. Method Not Allowed (GET)
	reqGet := httptest.NewRequest(http.MethodGet, "/api/telemetry", nil)
	rrGet := httptest.NewRecorder()
	handler.HandleIngest(rrGet, reqGet)
	if rrGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405 on GET /api/telemetry, got %d", rrGet.Code)
	}

	// 2. Invalid JSON payload
	reqBadJSON := httptest.NewRequest(http.MethodPost, "/api/telemetry", strings.NewReader("{invalid-json"))
	rrBadJSON := httptest.NewRecorder()
	handler.HandleIngest(rrBadJSON, reqBadJSON)
	if rrBadJSON.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 on invalid JSON, got %d", rrBadJSON.Code)
	}

	// 3. Valid JSON telemetry payload
	validJSON := `{
		"origin": "SENSOR-PRESSURE-01",
		"level": "WARNING",
		"category": "HARDWARE",
		"message": "High pipeline pressure detected"
	}`
	reqValid := httptest.NewRequest(http.MethodPost, "/api/telemetry", strings.NewReader(validJSON))
	rrValid := httptest.NewRecorder()
	handler.HandleIngest(rrValid, reqValid)

	if rrValid.Code != http.StatusOK {
		t.Errorf("Expected 200 on valid telemetry POST, got %d", rrValid.Code)
	}

	if len(processor.events) != 1 {
		t.Fatalf("Expected 1 event passed to processor, got %d", len(processor.events))
	}
	if processor.events[0].identifier != "SENSOR-PRESSURE-01" {
		t.Errorf("Expected identifier SENSOR-PRESSURE-01, got %s", processor.events[0].identifier)
	}
	if processor.events[0].event.Message != "High pipeline pressure detected" {
		t.Errorf("Event message mismatch: %s", processor.events[0].event.Message)
	}

	// 4. Missing required field 'origin'
	missingOriginJSON := `{
		"level": "WARNING",
		"category": "HARDWARE",
		"message": "High pipeline pressure detected"
	}`
	reqMissingOrigin := httptest.NewRequest(http.MethodPost, "/api/telemetry", strings.NewReader(missingOriginJSON))
	rrMissingOrigin := httptest.NewRecorder()
	handler.HandleIngest(rrMissingOrigin, reqMissingOrigin)
	if rrMissingOrigin.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 when 'origin' is missing, got %d", rrMissingOrigin.Code)
	}
}
