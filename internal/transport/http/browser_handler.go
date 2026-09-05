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
// File: internal/transport/http/browser_handler.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package http

import (
	"encoding/json"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
)

// BrowserOpenerFunc defines the function signature for launching external URLs in the OS default browser.
type BrowserOpenerFunc func(targetURL string) error

func defaultBrowserOpener(targetURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", targetURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", targetURL)
	case "darwin":
		cmd = exec.Command("open", targetURL)
	default:
		cmd = exec.Command("xdg-open", targetURL)
	}
	return cmd.Start()
}

// BrowserHandler handles requests to open external HTTP/HTTPS links in the host operating system browser.
type BrowserHandler struct {
	opener BrowserOpenerFunc
}

// NewBrowserHandler creates a new BrowserHandler with default OS command execution.
func NewBrowserHandler() *BrowserHandler {
	return &BrowserHandler{opener: defaultBrowserOpener}
}

// NewBrowserHandlerWithOpener creates a BrowserHandler with an injected opener (for testing).
func NewBrowserHandlerWithOpener(opener BrowserOpenerFunc) *BrowserHandler {
	return &BrowserHandler{opener: opener}
}

// HandleOpenExternal opens an external HTTP/HTTPS URL in the operating system's default browser.
func (h *BrowserHandler) HandleOpenExternal(w http.ResponseWriter, r *http.Request) {
	targetURL := strings.TrimSpace(r.URL.Query().Get("url"))
	if targetURL == "" {
		targetURL = strings.TrimSpace(r.FormValue("url"))
	}

	// Security: Only allow http:// or https:// URLs
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		http.Error(w, "URL inválida", http.StatusBadRequest)
		return
	}

	log.Printf("[BROWSER] Opening external URL in default browser: %s", targetURL)

	if err := h.opener(targetURL); err != nil {
		log.Printf("[BROWSER] Failed to open external URL '%s': %v", targetURL, err)
		http.Error(w, "Falha ao abrir navegador", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}
