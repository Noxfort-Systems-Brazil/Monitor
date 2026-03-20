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
// File: internal/transport/http/device_handler.go
// Author: Gabriel Moraes
// Date: 2026-01-19

package http

import (
	"html/template"
	"log"
	"net/http"
	"noxfort-monitor-server/internal/appdir"

	"noxfort-monitor-server/internal/domain"
)

// DeviceHandler manages the CRUD operations for monitored systems.
type DeviceHandler struct {
	deviceRepo domain.DeviceRepository
}

// NewDeviceHandler initializes the handler.
// Refactored: Removed unused SettingsRepository dependency.
func NewDeviceHandler(dRepo domain.DeviceRepository) *DeviceHandler {
	return &DeviceHandler{
		deviceRepo: dRepo,
	}
}

// ServePage renders the management list.
func (h *DeviceHandler) ServePage(w http.ResponseWriter, r *http.Request) {
	devices, err := h.deviceRepo.GetAllDevices()
	if err != nil {
		log.Printf("[DEVICES] Error fetching devices: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Render the management view
	tmpl, err := template.ParseFiles(
		appdir.Path("web/templates/layout.html"),
		appdir.Path("web/templates/devices.html"),
	)
	if err != nil {
		log.Printf("[DEVICES] Template error: %v", err)
		http.Error(w, "Template Error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":   "System Management",
		"Devices": devices, // Passes the []domain.Device list directly
	}

	tmpl.Execute(w, data)
}

// HandleDelete removes a system from monitoring.
func (h *DeviceHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	// We delete by Identifier (string), passed in the query param 'id'
	// URL Example: /devices/delete?id=synapse
	identifier := r.URL.Query().Get("id")

	if identifier != "" {
		log.Printf("[DEVICES] Requesting deletion for system: %s", identifier)
		if err := h.deviceRepo.DeleteDevice(identifier); err != nil {
			log.Printf("[DEVICES] Failed to delete %s: %v", identifier, err)
		} else {
			log.Printf("[DEVICES] System %s removed successfully.", identifier)
		}
	} else {
		log.Println("[DEVICES] Delete request missing 'id' parameter.")
	}

	// Refresh the list
	http.Redirect(w, r, "/devices", http.StatusSeeOther)
}
