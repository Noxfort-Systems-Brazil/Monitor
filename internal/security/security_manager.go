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
// File: internal/security/security_manager.go
// Author: Gabriel Moraes
// Date: 2026-09-03
// Modified: 2026-09-04 (SOLID Refactor)

package security

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"noxfort-monitor-server/internal/domain"
)

// SecurityManager coordinates user registration, authentication, and session management.
type SecurityManager struct {
	userRepo     domain.UserRepository
	hasher       PasswordHasher
	sessionStore SessionStore
	auditRepo    domain.AuditRepository
}

// NewSecurityManager creates a new SecurityManager instance with default PBKDF2 hasher and memory store.
func NewSecurityManager(userRepo domain.UserRepository) *SecurityManager {
	return NewSecurityManagerWithCustom(userRepo, defaultHasher, NewMemorySessionStore(SessionTTL))
}

// NewSecurityManagerWithCustom creates a SecurityManager with injected PasswordHasher and SessionStore.
func NewSecurityManagerWithCustom(userRepo domain.UserRepository, hasher PasswordHasher, sessionStore SessionStore) *SecurityManager {
	return &SecurityManager{
		userRepo:     userRepo,
		hasher:       hasher,
		sessionStore: sessionStore,
	}
}

// SetAuditRepository sets the audit repository for recording security events.
func (sm *SecurityManager) SetAuditRepository(repo domain.AuditRepository) {
	sm.auditRepo = repo
}

func (sm *SecurityManager) logAudit(username, action, details, ip string) {
	if sm.auditRepo != nil {
		_ = sm.auditRepo.SaveSecurityAuditLog(&domain.SecurityAuditLog{
			Username:  username,
			Action:    action,
			Details:   details,
			IPAddress: ip,
			CreatedAt: time.Now(),
		})
	}
}

// EnsureSuperuser guarantees that the designated superuser account exists with ADMIN role.
func (sm *SecurityManager) EnsureSuperuser() error {
	return BootstrapSuperuser(sm.userRepo, sm.hasher)
}

// HasAdmin checks whether there is already an administrator registered in the system.
func (sm *SecurityManager) HasAdmin() (bool, error) {
	count, err := sm.userRepo.CountAdmins()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// RegisterWithRole creates a new user account with an explicit role (ADMIN or OPERATOR).
func (sm *SecurityManager) RegisterWithRole(username, password, role string) (*domain.User, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, errors.New("Nome de usuário e senha são obrigatórios")
	}

	if len(password) < 4 {
		return nil, errors.New("A senha deve conter pelo menos 4 caracteres")
	}

	existing, _ := sm.userRepo.GetByUsername(username)
	if existing != nil {
		return nil, errors.New("Nome de usuário já está em uso")
	}

	if role != domain.RoleAdmin && role != domain.RoleOperator {
		role = domain.RoleOperator
	}

	hash, err := sm.hasher.Hash(password)
	if err != nil {
		return nil, fmt.Errorf("falha ao criptografar senha: %w", err)
	}

	user := &domain.User{
		Username:     username,
		PasswordHash: hash,
		Role:         role,
		CreatedAt:    time.Now(),
	}

	if err := sm.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("falha ao salvar usuário: %w", err)
	}

	sm.logAudit(username, "USER_CREATE", fmt.Sprintf("Usuário cadastrado com perfil %s", role), "")
	log.Printf("[SECURITY] Novo usuário '%s' cadastrado com perfil %s.", username, role)
	return user, nil
}

// Register creates a new user account following the business rule:
// - If no admin exists in the system, the user is created as ADMIN.
// - All subsequent registrations become OPERATOR.
func (sm *SecurityManager) Register(username, password string) (*domain.User, error) {
	adminCount, err := sm.userRepo.CountAdmins()
	if err != nil {
		return nil, fmt.Errorf("falha ao verificar administradores: %w", err)
	}

	role := domain.RoleOperator
	if adminCount == 0 {
		role = domain.RoleAdmin
		log.Printf("[SECURITY] Primeiro usuário do sistema '%s' definido como ADMINISTRADOR.", username)
	}

	return sm.RegisterWithRole(username, password, role)
}


// Authenticate verifies user credentials against the database.
func (sm *SecurityManager) Authenticate(username, password string) (*domain.User, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		sm.logAudit(username, "AUTH_LOGIN_FAILED", "Tentativa de login com campos vazios", "")
		return nil, errors.New("Usuário e senha são obrigatórios")
	}

	user, err := sm.userRepo.GetByUsername(username)
	if err != nil || user == nil {
		sm.logAudit(username, "AUTH_LOGIN_FAILED", "Usuário não encontrado", "")
		return nil, errors.New("Usuário ou senha incorretos")
	}

	if !sm.hasher.Verify(password, user.PasswordHash) {
		sm.logAudit(username, "AUTH_LOGIN_FAILED", "Senha incorreta", "")
		return nil, errors.New("Usuário ou senha incorretos")
	}

	sm.logAudit(user.Username, "AUTH_LOGIN_SUCCESS", fmt.Sprintf("Login efetuado com perfil %s", user.Role), "")
	log.Printf("[SECURITY] Usuário '%s' (%s) logado com sucesso.", user.Username, user.Role)
	return user, nil
}

// CreateSession generates a new session token via the configured SessionStore.
func (sm *SecurityManager) CreateSession(username, role string) string {
	return sm.sessionStore.CreateSession(username, role)
}

// ValidateSession verifies if a session token is active and valid via the configured SessionStore.
func (sm *SecurityManager) ValidateSession(token string) (string, string, bool) {
	return sm.sessionStore.ValidateSession(token)
}

// RevokeSession invalidates an active session token via the configured SessionStore.
func (sm *SecurityManager) RevokeSession(token string) {
	sm.sessionStore.RevokeSession(token)
}

// ListUsers returns all users in the system.
func (sm *SecurityManager) ListUsers() ([]*domain.User, error) {
	return sm.userRepo.List()
}

// DeleteUser removes an operator account.
func (sm *SecurityManager) DeleteUser(username string) error {
	username = strings.TrimSpace(username)
	user, err := sm.userRepo.GetByUsername(username)
	if err != nil || user == nil {
		return errors.New("Usuário não encontrado")
	}

	if user.Role == domain.RoleAdmin {
		return errors.New("Não é permitido excluir o Administrador do sistema")
	}

	if err := sm.userRepo.DeleteByUsername(username); err != nil {
		return err
	}
	sm.logAudit(username, "USER_DELETE", "Conta de operador excluída", "")
	return nil
}
