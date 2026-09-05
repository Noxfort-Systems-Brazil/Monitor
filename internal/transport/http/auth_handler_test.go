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
// File: internal/transport/http/auth_handler_test.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"noxfort-monitor-server/internal/domain"
)

type mockAuthService struct {
	users    map[string]*domain.User
	sessions map[string]struct {
		username string
		role     string
	}
}

func newMockAuthService() *mockAuthService {
	return &mockAuthService{
		users: map[string]*domain.User{
			"admin_user": {
				Username: "admin_user",
				Role:     domain.RoleAdmin,
			},
			"op_user": {
				Username: "op_user",
				Role:     domain.RoleOperator,
			},
		},
		sessions: make(map[string]struct {
			username string
			role     string
		}),
	}
}

func (m *mockAuthService) Authenticate(username, password string) (*domain.User, error) {
	u, ok := m.users[username]
	if !ok || password != "secret" {
		return nil, errors.New("credenciais inválidas")
	}
	return u, nil
}

func (m *mockAuthService) CreateSession(username, role string) string {
	token := "token-" + username
	m.sessions[token] = struct {
		username string
		role     string
	}{username: username, role: role}
	return token
}

func (m *mockAuthService) ValidateSession(token string) (string, string, bool) {
	s, ok := m.sessions[token]
	if !ok {
		return "", "", false
	}
	return s.username, s.role, true
}

func (m *mockAuthService) RevokeSession(token string) {
	delete(m.sessions, token)
}

func TestAuthHandler_LoginAndLogout(t *testing.T) {
	authService := newMockAuthService()
	handler := NewAuthHandler(authService)

	// 1. Invalid login attempt
	badForm := url.Values{}
	badForm.Set("username", "admin_user")
	badForm.Set("password", "wrongpassword")

	reqBad := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(badForm.Encode()))
	reqBad.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rrBad := httptest.NewRecorder()

	handler.HandleLogin(rrBad, reqBad)
	if rrBad.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 on bad credentials, got %d", rrBad.Code)
	}

	// 2. Successful login via form
	loginForm := url.Values{}
	loginForm.Set("username", "admin_user")
	loginForm.Set("password", "secret")

	reqLogin := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(loginForm.Encode()))
	reqLogin.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rrLogin := httptest.NewRecorder()

	handler.HandleLogin(rrLogin, reqLogin)
	if rrLogin.Code != http.StatusOK {
		t.Fatalf("Expected 200 on valid credentials, got %d", rrLogin.Code)
	}

	var loginRes map[string]interface{}
	if err := json.NewDecoder(rrLogin.Body).Decode(&loginRes); err != nil {
		t.Fatalf("Failed to decode login JSON: %v", err)
	}

	token, ok := loginRes["token"].(string)
	if !ok || token == "" {
		t.Fatalf("Expected login response to contain non-empty token")
	}

	// Verify session cookie was set
	cookies := rrLogin.Result().Cookies()
	var authCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == sessionCookieName {
			authCookie = c
			break
		}
	}
	if authCookie == nil || authCookie.Value != token {
		t.Fatalf("Expected session cookie '%s' to be set with token '%s'", sessionCookieName, token)
	}

	// 3. Check auth status
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	reqStatus.AddCookie(authCookie)
	rrStatus := httptest.NewRecorder()

	handler.HandleStatus(rrStatus, reqStatus)
	if rrStatus.Code != http.StatusOK {
		t.Fatalf("Expected 200 on status check, got %d", rrStatus.Code)
	}

	var statusRes map[string]interface{}
	_ = json.NewDecoder(rrStatus.Body).Decode(&statusRes)
	if statusRes["authenticated"] != true || statusRes["username"] != "admin_user" {
		t.Fatalf("Expected authenticated=true for admin_user, got: %v", statusRes)
	}

	// 4. Logout
	reqLogout := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	reqLogout.Header.Set("Accept", "application/json")
	reqLogout.AddCookie(authCookie)
	rrLogout := httptest.NewRecorder()

	handler.HandleLogout(rrLogout, reqLogout)
	if rrLogout.Code != http.StatusOK {
		t.Fatalf("Expected 200 on JSON logout, got %d", rrLogout.Code)
	}

	// Verify session is now invalid
	username, role, isAuth := authService.ValidateSession(token)
	if isAuth {
		t.Fatalf("Expected session to be revoked, but user=%s role=%s was returned", username, role)
	}
}

func TestAuthHandler_TokenExtraction(t *testing.T) {
	authService := newMockAuthService()
	handler := NewAuthHandler(authService)

	token := authService.CreateSession("admin_user", domain.RoleAdmin)

	// 1. Authorization Bearer header
	reqBearer := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	reqBearer.Header.Set("Authorization", "Bearer "+token)
	user, role, ok := handler.GetSessionUser(reqBearer)
	if !ok || user != "admin_user" || role != domain.RoleAdmin {
		t.Errorf("Failed to validate session from Bearer header: user=%s, role=%s, ok=%v", user, role, ok)
	}

	// 2. X-Session-Token header
	reqCustom := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	reqCustom.Header.Set("X-Session-Token", token)
	user, role, ok = handler.GetSessionUser(reqCustom)
	if !ok || user != "admin_user" || role != domain.RoleAdmin {
		t.Errorf("Failed to validate session from X-Session-Token: user=%s, role=%s, ok=%v", user, role, ok)
	}

	// 3. URL Query parameter (?token=...)
	reqQuery := httptest.NewRequest(http.MethodGet, "/api/auth/status?token="+token, nil)
	user, role, ok = handler.GetSessionUser(reqQuery)
	if !ok || user != "admin_user" || role != domain.RoleAdmin {
		t.Errorf("Failed to validate session from URL query: user=%s, role=%s, ok=%v", user, role, ok)
	}
}
