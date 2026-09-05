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
// File: internal/transport/http/telemetry_handler.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"noxfort-monitor-server/internal/monitor"
	"noxfort-monitor-server/internal/protocol"
)

// TelemetryHandler processes HTTP REST telemetry ingest payloads from IoT sensors and systems.
type TelemetryHandler struct {
	processor monitor.EventProcessor
}

// NewTelemetryHandler creates a new TelemetryHandler instance.
func NewTelemetryHandler(processor monitor.EventProcessor) *TelemetryHandler {
	return &TelemetryHandler{processor: processor}
}

// HandleIngest receives raw telemetry event data via HTTP POST.
func (h *TelemetryHandler) HandleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Validate and decode through single source of truth (protocol.DecodePayload)
	event, err := protocol.DecodePayload(body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid Payload: %v", err), http.StatusBadRequest)
		return
	}

	if h.processor != nil {
		h.processor.ProcessEvent(event.Origin, event)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "received"})
}
