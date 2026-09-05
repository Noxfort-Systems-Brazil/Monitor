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
// File: internal/transport/http/user_handler_test.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"noxfort-monitor-server/internal/domain"
	"noxfort-monitor-server/internal/security"
)

type mockUserRepoForHandler struct {
	users map[string]*domain.User
}

func (m *mockUserRepoForHandler) GetByUsername(username string) (*domain.User, error) {
	if u, ok := m.users[username]; ok {
		return u, nil
	}
	return nil, nil
}

func (m *mockUserRepoForHandler) List() ([]*domain.User, error) {
	var l []*domain.User
	for _, u := range m.users {
		l = append(l, u)
	}
	return l, nil
}

func (m *mockUserRepoForHandler) Create(u *domain.User) error {
	m.users[u.Username] = u
	return nil
}

func (m *mockUserRepoForHandler) Delete(id int64) error {
	return nil
}

func (m *mockUserRepoForHandler) DeleteByUsername(username string) error {
	delete(m.users, username)
	return nil
}

func (m *mockUserRepoForHandler) CountAdmins() (int, error) {
	count := 0
	for _, u := range m.users {
		if u.Role == domain.RoleAdmin {
			count++
		}
	}
	return count, nil
}

func TestUserHandler_CRUDAndAuthorization(t *testing.T) {
	repo := &mockUserRepoForHandler{users: make(map[string]*domain.User)}
	sm := security.NewSecurityManager(repo)
	if err := sm.EnsureSuperuser(); err != nil {
		t.Fatalf("Failed to initialize superuser: %v", err)
	}
	authHandler := NewAuthHandler(sm)
	userHandler := NewUserHandler(sm, sm)

	// 1. Unauthenticated attempt to register must be rejected (403)
	formOp := url.Values{}
	formOp.Set("username", "plant_op")
	formOp.Set("password", "operatorpass")

	reqUnauth := httptest.NewRequest(http.MethodPost, "/api/users/create", strings.NewReader(formOp.Encode()))
	reqUnauth.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rrUnauth := httptest.NewRecorder()

	userHandler.HandleCreateUser(rrUnauth, reqUnauth)
	if rrUnauth.Code != http.StatusForbidden {
		t.Fatalf("Expected unauthenticated user creation to return 403, got %d", rrUnauth.Code)
	}

	// 2. Login with configured Superuser credentials via AuthHandler
	loginForm := url.Values{}
	loginForm.Set("username", security.SuperuserUsername)
	loginForm.Set("password", security.SuperuserPassword)

	reqLogin := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(loginForm.Encode()))
	reqLogin.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rrLogin := httptest.NewRecorder()

	authHandler.HandleLogin(rrLogin, reqLogin)
	if rrLogin.Code != http.StatusOK {
		t.Fatalf("Expected superuser login status 200, got %d. Body: %s", rrLogin.Code, rrLogin.Body.String())
	}

	// Extract admin session cookie
	cookies := rrLogin.Result().Cookies()
	var adminCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == sessionCookieName {
			adminCookie = c
			break
		}
	}
	if adminCookie == nil {
		t.Fatalf("Expected session cookie to be set upon superuser login")
	}

	// 3. Authenticated Admin creates new Operator
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/users/create", strings.NewReader(formOp.Encode()))
	reqCreate.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqCreate.AddCookie(adminCookie)
	rrCreate := httptest.NewRecorder()

	userHandler.HandleCreateUser(rrCreate, reqCreate)
	if rrCreate.Code != http.StatusOK {
		t.Fatalf("Expected operator creation to return 200, got %d. Body: %s", rrCreate.Code, rrCreate.Body.String())
	}

	var resOp map[string]interface{}
	_ = json.NewDecoder(rrCreate.Body).Decode(&resOp)
	if resOp["role"] != domain.RoleOperator {
		t.Fatalf("Expected created user to have role %s, got %v", domain.RoleOperator, resOp["role"])
	}

	// 4. Operator logs in successfully via AuthHandler
	opLoginForm := url.Values{}
	opLoginForm.Set("username", "plant_op")
	opLoginForm.Set("password", "operatorpass")

	reqOpLogin := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(opLoginForm.Encode()))
	reqOpLogin.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rrOpLogin := httptest.NewRecorder()

	authHandler.HandleLogin(rrOpLogin, reqOpLogin)
	if rrOpLogin.Code != http.StatusOK {
		t.Fatalf("Expected operator login 200, got %d", rrOpLogin.Code)
	}

	// Extract operator session cookie
	var opCookie *http.Cookie
	for _, c := range rrOpLogin.Result().Cookies() {
		if c.Name == sessionCookieName {
			opCookie = c
			break
		}
	}
	if opCookie == nil {
		t.Fatalf("Expected operator session cookie")
	}

	// Operator cannot create users (must be 403)
	reqOpCreate := httptest.NewRequest(http.MethodPost, "/api/users/create", strings.NewReader(formOp.Encode()))
	reqOpCreate.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqOpCreate.AddCookie(opCookie)
	rrOpCreate := httptest.NewRecorder()

	userHandler.HandleCreateUser(rrOpCreate, reqOpCreate)
	if rrOpCreate.Code != http.StatusForbidden {
		t.Fatalf("Expected operator user creation to return 403, got %d", rrOpCreate.Code)
	}

	// 5. Admin lists users
	reqList := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	reqList.AddCookie(adminCookie)
	rrList := httptest.NewRecorder()

	userHandler.HandleList(rrList, reqList)
	if rrList.Code != http.StatusOK {
		t.Fatalf("Expected list status 200, got %d", rrList.Code)
	}

	// 6. Admin deletes operator
	delForm := url.Values{}
	delForm.Set("username", "plant_op")

	reqDel := httptest.NewRequest(http.MethodPost, "/api/users/delete", strings.NewReader(delForm.Encode()))
	reqDel.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqDel.AddCookie(adminCookie)
	rrDel := httptest.NewRecorder()

	userHandler.HandleDelete(rrDel, reqDel)
	if rrDel.Code != http.StatusOK {
		t.Fatalf("Expected delete status 200, got %d. Body: %s", rrDel.Code, rrDel.Body.String())
	}

	// 7. Admin tries to delete superuser (must be blocked)
	delSelfForm := url.Values{}
	delSelfForm.Set("username", security.SuperuserUsername)

	reqDelSelf := httptest.NewRequest(http.MethodPost, "/api/users/delete", strings.NewReader(delSelfForm.Encode()))
	reqDelSelf.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqDelSelf.AddCookie(adminCookie)
	rrDelSelf := httptest.NewRecorder()

	userHandler.HandleDelete(rrDelSelf, reqDelSelf)
	if rrDelSelf.Code != http.StatusBadRequest {
		t.Fatalf("Expected deleting admin to return 400, got %d", rrDelSelf.Code)
	}
}

func TestUserHandler_TokenInHeadersAndQuery(t *testing.T) {
	repo := &mockUserRepoForHandler{users: make(map[string]*domain.User)}
	sm := security.NewSecurityManager(repo)
	if err := sm.EnsureSuperuser(); err != nil {
		t.Fatalf("Failed to initialize superuser: %v", err)
	}
	handler := NewUserHandler(sm, sm)
	token := sm.CreateSession(security.SuperuserUsername, domain.RoleAdmin)

	// 1. Validate session via Authorization Bearer header
	reqBearer := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	reqBearer.Header.Set("Authorization", "Bearer "+token)
	username, role, isAuth := handler.GetSessionUser(reqBearer)
	if !isAuth || username != security.SuperuserUsername || role != domain.RoleAdmin {
		t.Fatalf("Expected valid session via Bearer header, got user=%q, role=%q, isAuth=%v", username, role, isAuth)
	}

	// 2. Validate session via X-Session-Token header
	reqCustom := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	reqCustom.Header.Set("X-Session-Token", token)
	username, role, isAuth = handler.GetSessionUser(reqCustom)
	if !isAuth || username != security.SuperuserUsername || role != domain.RoleAdmin {
		t.Fatalf("Expected valid session via X-Session-Token header, got user=%q, role=%q, isAuth=%v", username, role, isAuth)
	}

	// 3. Validate session via URL query param (?token=...)
	reqQuery := httptest.NewRequest(http.MethodGet, "/api/users?token="+token, nil)
	username, role, isAuth = handler.GetSessionUser(reqQuery)
	if !isAuth || username != security.SuperuserUsername || role != domain.RoleAdmin {
		t.Fatalf("Expected valid session via query parameter, got user=%q, role=%q, isAuth=%v", username, role, isAuth)
	}
}
