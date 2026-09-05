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
// File: internal/transport/http/auth_handler.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package http

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"

	"noxfort-monitor-server/internal/appdir"
	"noxfort-monitor-server/internal/domain"
)

// AuthService defines the authentication and session management contract required by HTTP transport.
type AuthService interface {
	Authenticate(username, password string) (*domain.User, error)
	CreateSession(username, role string) string
	ValidateSession(token string) (string, string, bool)
	RevokeSession(token string)
}

// AuthHandler manages authentication, login, logout, and session state.
type AuthHandler struct {
	authService AuthService
}

// NewAuthHandler creates a new AuthHandler instance.
func NewAuthHandler(authService AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// ExtractToken retrieves the session token from cookies, Authorization headers, or custom headers.
func (h *AuthHandler) ExtractToken(r *http.Request) string {
	return ExtractSessionToken(r)
}

// GetSessionUser checks the request cookie and headers, returning the authenticated username, role, and true if valid.
func (h *AuthHandler) GetSessionUser(r *http.Request) (string, string, bool) {
	token := ExtractSessionToken(r)
	if token == "" {
		return "", "", false
	}
	return h.authService.ValidateSession(token)
}

// ServeLogin renders the login page.
func (h *AuthHandler) ServeLogin(w http.ResponseWriter, r *http.Request) {
	if _, _, isAuth := h.GetSessionUser(r); isAuth {
		Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	tmpl, err := template.ParseFiles(appdir.Path("web/templates/login.html"))
	if err != nil {
		log.Printf("[AUTH] Template error (login): %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	_ = tmpl.Execute(w, nil)
}

// ServeRegister redirects unauthenticated visitors to /login, or authenticated admins to /users.
func (h *AuthHandler) ServeRegister(w http.ResponseWriter, r *http.Request) {
	_, role, isAuth := h.GetSessionUser(r)
	if isAuth && role == domain.RoleAdmin {
		Redirect(w, r, "/users", http.StatusSeeOther)
		return
	}
	Redirect(w, r, "/login", http.StatusSeeOther)
}

// HandleLogin authenticates credentials and sets an HTTP-only session cookie.
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	username, password := parseCredentials(r)

	w.Header().Set("Content-Type", "application/json")

	user, err := h.authService.Authenticate(username, password)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	token := h.authService.CreateSession(user.Username, user.Role)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"token":    token,
		"username": user.Username,
		"role":     user.Role,
	})
}

// HandleLogout clears the user session and redirects to /login.
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if token := ExtractSessionToken(r); token != "" {
		h.authService.RevokeSession(token)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
		return
	}

	Redirect(w, r, "/login", http.StatusSeeOther)
}

// HandleStatus returns JSON info about the current session.
func (h *AuthHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	username, role, isAuth := h.GetSessionUser(r)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"authenticated": isAuth,
		"username":      username,
		"role":          role,
	})
}
