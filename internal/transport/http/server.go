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
// File: internal/transport/http/server.go
// Author: Gabriel Moraes
// Date: 2026-01-19

package http

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"noxfort-monitor-server/internal/appdir"
	"noxfort-monitor-server/internal/domain"
	"noxfort-monitor-server/internal/monitor"
)

// Server is the HTTP Router and Orchestrator.
type Server struct {
	addr string

	// Modular Handlers
	dashboardHandler *DashboardHandler
	deviceHandler    *DeviceHandler
	contactHandler   *ContactHandler
	settingsHandler  *SettingsHandler

	// Dependencies for API Ingest
	stateManager *monitor.StateManager
}

// NewServer initializes the HTTP server and its sub-handlers.
func NewServer(
	addr string,
	dRepo domain.DeviceRepository,
	tRepo domain.TelemetryRepository,
	cRepo domain.ContactRepository,
	sRepo domain.SettingsRepository,
	sm *monitor.StateManager,
	alertService *monitor.AlertService,
) *Server {
	return &Server{
		addr: addr,
		// Refactored: Passed tRepo to DashboardHandler to allow fetching incidents
		dashboardHandler: NewDashboardHandler(dRepo, tRepo),
		deviceHandler:    NewDeviceHandler(dRepo),
		contactHandler:   NewContactHandler(cRepo),
		settingsHandler:  NewSettingsHandler(sRepo, alertService),
		stateManager:     sm,
	}
}

// Run configures the routes and starts the listening loop.
func (s *Server) Run() error {
	mux := http.NewServeMux()

	// 1. Static Assets
	fs := http.FileServer(http.Dir(appdir.Path("web/static")))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// 2. Systems Monitor (Home)
	mux.HandleFunc("/", s.dashboardHandler.ServePage)

	// 3. System Management
	mux.HandleFunc("/devices", s.deviceHandler.ServePage)
	mux.HandleFunc("/devices/delete", s.deviceHandler.HandleDelete)

	// 4. Response Team (Contacts)
	mux.HandleFunc("/contacts", s.contactHandler.ServePage)
	mux.HandleFunc("/contacts/create", s.contactHandler.HandleCreate)
	mux.HandleFunc("/contacts/delete", s.contactHandler.HandleDelete)

	// 5. Settings
	mux.HandleFunc("/settings", s.settingsHandler.ServePage)
	mux.HandleFunc("/settings/save", s.settingsHandler.HandleSave)
	mux.HandleFunc("/settings/test", s.settingsHandler.HandleTest)
	mux.HandleFunc("/settings/test-telegram", s.settingsHandler.HandleTestTelegram)

	// 6. IoT Telemetry API (HTTP POST Ingest)
	mux.HandleFunc("/api/telemetry", s.handleTelemetryIngest)

	server := &http.Server{
		Addr:         s.addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Printf("🌍 HTTP Server listening on %s", s.addr)
	return server.ListenAndServe()
}

// handleTelemetryIngest receives raw data from systems via HTTP POST.
func (s *Server) handleTelemetryIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var event domain.IncomingEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "Invalid JSON Payload", http.StatusBadRequest)
		return
	}

	// Use the same logic as MQTT: Origin is our Identifier
	s.stateManager.ProcessEvent(event.Origin, &event)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "received"})
}
