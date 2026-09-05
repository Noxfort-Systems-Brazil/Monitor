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
// File: internal/transport/http/middleware.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package http

import (
	"net/http"
	"strings"

	"noxfort-monitor-server/internal/domain"
)

// AuthSessionInspector verifies session credentials and can serve the login page.
type AuthSessionInspector interface {
	GetSessionUser(r *http.Request) (string, string, bool)
	ServeLogin(w http.ResponseWriter, r *http.Request)
}

// AuthMiddleware intercepts requests to enforce authentication and operator read-only restrictions (RBAC).
type AuthMiddleware struct {
	authInspector AuthSessionInspector
}

// NewAuthMiddleware creates a new AuthMiddleware instance.
func NewAuthMiddleware(authInspector AuthSessionInspector) *AuthMiddleware {
	return &AuthMiddleware{authInspector: authInspector}
}

// isPublicPath determines if a path can be accessed without an active session.
func isPublicPath(path string) bool {
	return strings.HasPrefix(path, "/static/") ||
		path == "/login" ||
		path == "/api/auth/login" ||
		path == "/api/auth/status" ||
		path == "/api/telemetry" ||
		path == "/api/open-external" ||
		strings.HasPrefix(path, "/api/window/")
}

// isMutatingRoute checks if the request represents a state modification action restricted to administrators.
func isMutatingRoute(r *http.Request, path string) bool {
	if path == "/api/auth/logout" || path == "/api/open-external" || strings.HasPrefix(path, "/api/window/") {
		return false
	}

	return r.Method == http.MethodPost ||
		path == "/devices/delete" ||
		strings.HasPrefix(path, "/contacts/create") ||
		strings.HasPrefix(path, "/contacts/update") ||
		strings.HasPrefix(path, "/contacts/delete") ||
		strings.HasPrefix(path, "/settings/save") ||
		strings.HasPrefix(path, "/settings/test")
}

// Wrap wraps an existing http.Handler with authentication and RBAC checks.
func (m *AuthMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// 1. Allow public endpoints
		if isPublicPath(path) {
			next.ServeHTTP(w, r)
			return
		}

		// 2. Verify active session
		_, role, isAuth := m.authInspector.GetSessionUser(r)
		if !isAuth {
			if strings.HasPrefix(path, "/api/") {
				http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
				return
			}
			// When unauthenticated access hits root path, serve login directly
			if path == "/" {
				m.authInspector.ServeLogin(w, r)
				return
			}
			Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// 3. Enforce Admin-only write restrictions (Operators are read-only)
		if role != domain.RoleAdmin && isMutatingRoute(r, path) {
			http.Error(w, "Acesso restrito: Operadores têm permissão somente leitura. Apenas o Administrador pode efetuar alterações.", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
