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
// File: internal/transport/http/auth_helper.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package http

import (
	"encoding/json"
	"net/http"
	"strings"
)

const sessionCookieName = "noxfort_session"

// ExtractSessionToken retrieves the session token from cookies, Authorization headers, or custom headers.
func ExtractSessionToken(r *http.Request) string {
	if cookie, err := r.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	if token := strings.TrimSpace(r.Header.Get("X-Session-Token")); token != "" {
		return token
	}
	if token := strings.TrimSpace(r.URL.Query().Get("token")); token != "" {
		return token
	}
	return ""
}

// parseCredentials extracts username and password from form-encoded or JSON payloads.
func parseCredentials(r *http.Request) (string, string) {
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	if username == "" && strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			username = strings.TrimSpace(req.Username)
			password = req.Password
		}
	}

	return username, password
}
