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
// File: internal/transport/http/tunnel_handler.go
// Author: Gabriel Moraes
// Date: 2026-09-04
// Modified: 2026-09-04 (SOLID: Depends on tunnel.Service interface)

package http

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"

	"noxfort-monitor-server/internal/appdir"
	"noxfort-monitor-server/internal/domain"
	"noxfort-monitor-server/internal/tunnel"
)

// TunnelHandler manages the Remote Access (Ngrok Tunnel) page and API endpoints.
// Adheres to Dependency Inversion Principle (DIP) by depending on the tunnel.Service abstraction.
type TunnelHandler struct {
	settingsRepo  domain.SettingsRepository
	tunnelService tunnel.Service
}

// NewTunnelHandler creates a new TunnelHandler instance with an injected tunnel service.
func NewTunnelHandler(settingsRepo domain.SettingsRepository, ts tunnel.Service) *TunnelHandler {
	if ts == nil {
		ts = tunnel.NewManager(nil, "8080")
	}
	return &TunnelHandler{
		settingsRepo:  settingsRepo,
		tunnelService: ts,
	}
}

// ServePage renders the dedicated Remote Access HTML page.
func (h *TunnelHandler) ServePage(w http.ResponseWriter, r *http.Request) {
	settings, err := h.settingsRepo.GetSettings()
	if err != nil {
		log.Printf("[REMOTE] Error loading settings: %v", err)
		http.Error(w, "Failed to load system settings", http.StatusInternalServerError)
		return
	}

	status := h.tunnelService.GetStatus()
	localIP := GetLocalIP()

	data := map[string]interface{}{
		"Title":       "Remote Access",
		"Settings":    settings,
		"Status":      status,
		"LocalIP":     localIP,
		"LocalPort":   "8080",
		"LocalURL":    "http://" + localIP + ":8080/api/telemetry",
		"BinaryFound": h.tunnelService.IsBinaryAvailable(),
	}

	tmpl, err := template.ParseFiles(
		appdir.Path("web/templates/layout.html"),
		appdir.Path("web/templates/remote.html"),
	)
	if err != nil {
		log.Printf("[REMOTE] Template rendering error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("[REMOTE] Template execution error: %v", err)
	}
}

// HandleStatus returns the current live status of the tunnel as JSON.
func (h *TunnelHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	status := h.tunnelService.GetStatus()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

// HandleSave updates the tunnel credentials in the database and restarts the tunnel if enabled.
func (h *TunnelHandler) HandleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid Form Data", http.StatusBadRequest)
		return
	}

	token := tunnel.CleanAuthToken(r.FormValue("ngrok_auth_token"))
	domainName := tunnel.CleanDomain(r.FormValue("ngrok_domain"))
	autoStart := (token != "")

	settings, err := h.settingsRepo.GetSettings()
	if err != nil {
		http.Error(w, "Failed to fetch settings", http.StatusInternalServerError)
		return
	}

	// Update ngrok specific fields - autoStart is always true when token is configured
	settings.NgrokAuthToken = token
	settings.NgrokDomain = domainName
	settings.NgrokEnabled = autoStart

	if err := h.settingsRepo.SaveSettings(settings); err != nil {
		log.Printf("[REMOTE] Failed to save settings: %v", err)
		http.Error(w, "Failed to save settings: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// If token present, start/restart tunnel automatically
	if token != "" {
		if err := h.tunnelService.Start(token, domainName); err != nil {
			log.Printf("[REMOTE] Failed to start tunnel after save: %v", err)
		}
	} else {
		_ = h.tunnelService.Stop()
	}

	// Support AJAX / JSON response
	if r.Header.Get("Accept") == "application/json" || r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Configurações salvas com sucesso. Túnel em inicialização.",
		})
		return
	}

	http.Redirect(w, r, "/remote?success=1", http.StatusSeeOther)
}

// HandleStart triggers tunnel startup on demand.
func (h *TunnelHandler) HandleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	settings, err := h.settingsRepo.GetSettings()
	if err != nil || settings.NgrokAuthToken == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Authtoken não configurado. Por favor, configure seu token primeiro.",
		})
		return
	}

	if err := h.tunnelService.Start(settings.NgrokAuthToken, settings.NgrokDomain); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Túnel iniciado com sucesso.",
	})
}

// HandleStop stops the tunnel on demand.
func (h *TunnelHandler) HandleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	_ = h.tunnelService.Stop()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Túnel finalizado com sucesso.",
	})
}
