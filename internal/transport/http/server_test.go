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
// File: internal/transport/http/server_test.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"noxfort-monitor-server/internal/domain"
	"noxfort-monitor-server/internal/security"
)

func TestAuthMiddleware_UnauthenticatedAccess(t *testing.T) {
	userRepo := &mockUserRepoForHandler{users: make(map[string]*domain.User)}
	secManager := security.NewSecurityManager(userRepo)
	server := NewServer("", nil, nil, nil, nil, nil, nil, secManager, nil, nil, nil)
	handler := server.Handler()

	// 1. Unauthenticated request to "/" must serve the login page directly (status 200)
	// without any intermediate 303 redirect that would show "See Other" in desktop webviews.
	reqRoot := httptest.NewRequest(http.MethodGet, "/", nil)
	recRoot := httptest.NewRecorder()
	handler.ServeHTTP(recRoot, reqRoot)

	if recRoot.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on unauthenticated GET /, got %d", recRoot.Code)
	}
	bodyRoot := recRoot.Body.String()
	if !strings.Contains(bodyRoot, "loginForm") && !strings.Contains(bodyRoot, "Entrar") {
		t.Errorf("Expected login page content on GET /, got: %s", bodyRoot)
	}

	// 2. Unauthenticated request to protected route (e.g. "/devices") must redirect to /login
	// with smart Redirect containing auto-navigation script.
	reqDevices := httptest.NewRequest(http.MethodGet, "/devices", nil)
	recDevices := httptest.NewRecorder()
	handler.ServeHTTP(recDevices, reqDevices)

	if recDevices.Code != http.StatusSeeOther {
		t.Fatalf("Expected status 303 on unauthenticated GET /devices, got %d", recDevices.Code)
	}
	if loc := recDevices.Header().Get("Location"); loc != "/login" {
		t.Errorf("Expected Location '/login', got %q", loc)
	}
	bodyDevices := recDevices.Body.String()
	if !strings.Contains(bodyDevices, `window.location.replace("/login")`) {
		t.Errorf("Expected auto-redirect script in body, got: %s", bodyDevices)
	}

	// 3. Unauthenticated request to "/api/..." must return 401 Unauthorized
	reqAPI := httptest.NewRequest(http.MethodGet, "/api/devices", nil)
	recAPI := httptest.NewRecorder()
	handler.ServeHTTP(recAPI, reqAPI)

	if recAPI.Code != http.StatusUnauthorized {
		t.Fatalf("Expected status 401 on unauthenticated /api/..., got %d", recAPI.Code)
	}

	// 4. Unauthenticated request to /register must redirect to /login
	reqRegister := httptest.NewRequest(http.MethodGet, "/register", nil)
	recRegister := httptest.NewRecorder()
	handler.ServeHTTP(recRegister, reqRegister)
	if recRegister.Code != http.StatusSeeOther {
		t.Fatalf("Expected status 303 on unauthenticated GET /register, got %d", recRegister.Code)
	}

	// 5. Unauthenticated request to /api/users/create must return 401 Unauthorized
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/users/create", nil)
	recCreate := httptest.NewRecorder()
	handler.ServeHTTP(recCreate, reqCreate)
	if recCreate.Code != http.StatusUnauthorized {
		t.Fatalf("Expected status 401 on unauthenticated POST /api/users/create, got %d", recCreate.Code)
	}
}
