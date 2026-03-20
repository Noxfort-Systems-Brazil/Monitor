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
// File: internal/transport/http/dashboard_handler.go
// Author: Gabriel Moraes
// Date: 2026-01-19

package http

import (
	"html/template"
	"log"
	"net/http"
	"noxfort-monitor-server/internal/appdir"
	"time"

	"noxfort-monitor-server/internal/domain"
)

// DashboardHandler manages the main overview page.
type DashboardHandler struct {
	deviceRepo    domain.DeviceRepository
	telemetryRepo domain.TelemetryRepository
}

// NewDashboardHandler creates the handler with necessary dependencies.
func NewDashboardHandler(dRepo domain.DeviceRepository, tRepo domain.TelemetryRepository) *DashboardHandler {
	return &DashboardHandler{
		deviceRepo:    dRepo,
		telemetryRepo: tRepo,
	}
}

// ServePage renders the main dashboard HTML.
func (h *DashboardHandler) ServePage(w http.ResponseWriter, r *http.Request) {
	// 1. Fetch Devices (List View Logic)
	devices, err := h.deviceRepo.GetAllDevices()
	if err != nil {
		log.Printf("[DASHBOARD] Error fetching devices: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 2. Fetch Recent Incidents
	// FIX: Increased limit from 10 to 1000.
	// This ensures the "Recent Alerts" table has enough data to utilize the scrollbar
	// and show the complete history of incidents without cutting off older messages.
	incidents, err := h.telemetryRepo.GetRecentIncidents(1000)
	if err != nil {
		log.Printf("[DASHBOARD] Error fetching incidents: %v", err)
		incidents = []domain.IncomingEvent{}
	}

	// 3. Prepare View Data
	type DeviceView struct {
		ID         int64
		Name       string
		Identifier string
		LastSeen   string
		Status     string
		RowClass   string // Used for CSS styling (green/red)
	}

	var viewData []DeviceView
	threshold := 5 * time.Minute

	for _, d := range devices {
		// Note: We display all devices regardless of 'Enabled' status to match Device Manager.
		timeSince := time.Since(d.LastSeen)
		status := "ONLINE"
		rowClass := "table-success" // Bootstrap class for green row

		if timeSince > threshold {
			status = "OFFLINE"
			rowClass = "table-danger" // Bootstrap class for red row
		}

		viewData = append(viewData, DeviceView{
			ID:         d.ID,
			Name:       d.Name,
			Identifier: d.Identifier,
			LastSeen:   d.LastSeen.Format("15:04:05 02/01/2006"),
			Status:     status,
			RowClass:   rowClass,
		})
	}

	// 4. Render Template
	data := map[string]interface{}{
		"Title":           "System Overview",
		"Devices":         viewData,
		"RecentIncidents": incidents,
		"Now":             time.Now().Format(time.RFC3339),
	}

	tmpl, err := template.ParseFiles(
		appdir.Path("web/templates/layout.html"),
		appdir.Path("web/templates/dashboard.html"),
	)
	if err != nil {
		log.Printf("[DASHBOARD] Template error: %v", err)
		http.Error(w, "Template Error", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, data)
}
