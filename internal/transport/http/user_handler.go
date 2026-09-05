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
// File: internal/transport/http/user_handler.go
// Author: Gabriel Moraes
// Date: 2026-09-03
// Modified: 2026-09-04 (SOLID Refactor)

package http

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strings"

	"noxfort-monitor-server/internal/appdir"
	"noxfort-monitor-server/internal/domain"
)

// UserManagementService defines user account operations required by HTTP transport.
type UserManagementService interface {
	Register(username, password string) (*domain.User, error)
	RegisterWithRole(username, password, role string) (*domain.User, error)
	ListUsers() ([]*domain.User, error)
	DeleteUser(username string) error
}

// SessionValidator verifies active sessions and user roles.
type SessionValidator interface {
	ValidateSession(token string) (string, string, bool)
}

// UserHandler manages user accounts and operator administration.
type UserHandler struct {
	userService      UserManagementService
	sessionValidator SessionValidator
}

// NewUserHandler creates a new UserHandler instance.
func NewUserHandler(userService UserManagementService, sessionValidator SessionValidator) *UserHandler {
	return &UserHandler{
		userService:      userService,
		sessionValidator: sessionValidator,
	}
}

// GetSessionUser checks the request cookie and headers, returning username, role, and validity.
func (h *UserHandler) GetSessionUser(r *http.Request) (string, string, bool) {
	token := ExtractSessionToken(r)
	if token == "" {
		return "", "", false
	}
	return h.sessionValidator.ValidateSession(token)
}

// HandleCreateUser registers a new user (defaulting to OPERATOR). Only authenticated ADMIN users can perform this.
func (h *UserHandler) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	_, role, isAuth := h.GetSessionUser(r)
	if !isAuth || role != domain.RoleAdmin {
		http.Error(w, "Acesso não autorizado: apenas o Administrador pode cadastrar novos usuários", http.StatusForbidden)
		return
	}

	username, password := parseCredentials(r)

	targetRole := strings.TrimSpace(r.FormValue("role"))
	if targetRole != domain.RoleAdmin && targetRole != domain.RoleOperator {
		targetRole = domain.RoleOperator
	}

	w.Header().Set("Content-Type", "application/json")

	user, err := h.userService.RegisterWithRole(username, password, targetRole)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"username": user.Username,
		"role":     user.Role,
	})
}

// HandleRegister is an alias for HandleCreateUser, requiring admin privileges.
func (h *UserHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	h.HandleCreateUser(w, r)
}

// HandleList returns all registered users in JSON (ADMIN only).
func (h *UserHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	_, role, isAuth := h.GetSessionUser(r)
	if !isAuth || role != domain.RoleAdmin {
		http.Error(w, "Acesso não autorizado", http.StatusForbidden)
		return
	}

	users, err := h.userService.ListUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(users)
}

// HandleDelete removes an operator account (ADMIN only).
func (h *UserHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	_, role, isAuth := h.GetSessionUser(r)
	if !isAuth || role != domain.RoleAdmin {
		http.Error(w, "Acesso não autorizado: privilégios de administrador necessários", http.StatusForbidden)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	if username == "" {
		http.Error(w, "Nome de usuário é obrigatório", http.StatusBadRequest)
		return
	}

	if err := h.userService.DeleteUser(username); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	log.Printf("[USERS] User '%s' removed by Administrator.", username)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Usuário removido com sucesso",
	})
}

// ServePage renders the /users account management page (ADMIN only).
func (h *UserHandler) ServePage(w http.ResponseWriter, r *http.Request) {
	username, role, isAuth := h.GetSessionUser(r)
	if !isAuth {
		Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if role != domain.RoleAdmin {
		http.Error(w, "Acesso restrito: Apenas o Administrador pode acessar esta página.", http.StatusForbidden)
		return
	}

	users, err := h.userService.ListUsers()
	if err != nil {
		log.Printf("[USERS] Error listing users: %v", err)
	}

	tmpl, err := template.ParseFiles(
		appdir.Path("web/templates/layout.html"),
		appdir.Path("web/templates/users.html"),
	)
	if err != nil {
		log.Printf("[USERS] Template parse error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"CurrentUser": username,
		"Role":        role,
		"IsAdmin":     true,
		"Users":       users,
	}

	_ = tmpl.Execute(w, data)
}
