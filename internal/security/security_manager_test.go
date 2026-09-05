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
// File: internal/security/security_manager_test.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package security

import (
	"errors"
	"testing"

	"noxfort-monitor-server/internal/domain"
)

type mockUserRepo struct {
	users map[string]*domain.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*domain.User)}
}

func (m *mockUserRepo) GetByUsername(username string) (*domain.User, error) {
	u, ok := m.users[username]
	if !ok {
		return nil, errors.New("user not found")
	}
	return u, nil
}

func (m *mockUserRepo) List() ([]*domain.User, error) {
	var list []*domain.User
	for _, u := range m.users {
		list = append(list, u)
	}
	return list, nil
}

func (m *mockUserRepo) Create(user *domain.User) error {
	m.users[user.Username] = user
	return nil
}

func (m *mockUserRepo) Delete(id int64) error {
	for k, u := range m.users {
		if u.ID == id {
			delete(m.users, k)
			return nil
		}
	}
	return errors.New("user not found")
}

func (m *mockUserRepo) DeleteByUsername(username string) error {
	if _, ok := m.users[username]; !ok {
		return errors.New("user not found")
	}
	delete(m.users, username)
	return nil
}

func (m *mockUserRepo) CountAdmins() (int, error) {
	count := 0
	for _, u := range m.users {
		if u.Role == domain.RoleAdmin {
			count++
		}
	}
	return count, nil
}

func TestSingleAdminRuleAndOperator(t *testing.T) {
	repo := newMockUserRepo()
	sm := NewSecurityManager(repo)

	// First registration: must be ADMIN
	u1, err := sm.Register("superadmin", "strongpass1")
	if err != nil {
		t.Fatalf("Failed to register first user: %v", err)
	}
	if u1.Role != domain.RoleAdmin {
		t.Fatalf("Expected first user to have role %s, got %s", domain.RoleAdmin, u1.Role)
	}

	// Second registration: must be OPERATOR ("super somente ter um")
	u2, err := sm.Register("operator1", "strongpass2")
	if err != nil {
		t.Fatalf("Failed to register second user: %v", err)
	}
	if u2.Role != domain.RoleOperator {
		t.Fatalf("Expected second user to have role %s, got %s", domain.RoleOperator, u2.Role)
	}

	// Third registration: also OPERATOR
	u3, err := sm.Register("operator2", "strongpass3")
	if err != nil {
		t.Fatalf("Failed to register third user: %v", err)
	}
	if u3.Role != domain.RoleOperator {
		t.Fatalf("Expected third user to have role %s, got %s", domain.RoleOperator, u3.Role)
	}

	// Verify total admin count is strictly 1
	adminCount, _ := repo.CountAdmins()
	if adminCount != 1 {
		t.Fatalf("Expected strictly 1 admin, got %d", adminCount)
	}
}

func TestAuthenticationFlow(t *testing.T) {
	repo := newMockUserRepo()
	sm := NewSecurityManager(repo)

	_, _ = sm.Register("admin_user", "password123")

	// Successful authentication
	user, err := sm.Authenticate("admin_user", "password123")
	if err != nil || user == nil {
		t.Fatalf("Expected authentication to succeed, got %v", err)
	}
	if user.Role != domain.RoleAdmin {
		t.Fatalf("Expected role %s, got %s", domain.RoleAdmin, user.Role)
	}

	// Wrong password
	_, err = sm.Authenticate("admin_user", "wrongpassword")
	if err == nil {
		t.Fatalf("Expected authentication to fail on wrong password")
	}

	// Non-existent user
	_, err = sm.Authenticate("ghost", "password123")
	if err == nil {
		t.Fatalf("Expected authentication to fail for non-existent user")
	}
}

func TestAdminProtectionFromDeletion(t *testing.T) {
	repo := newMockUserRepo()
	sm := NewSecurityManager(repo)

	_, _ = sm.Register("main_admin", "pass1")
	_, _ = sm.Register("worker", "pass2")

	// Attempt to delete admin must fail
	err := sm.DeleteUser("main_admin")
	if err == nil {
		t.Fatalf("Expected deleting admin to fail, got nil")
	}

	// Deleting operator must succeed
	err = sm.DeleteUser("worker")
	if err != nil {
		t.Fatalf("Expected deleting operator to succeed, got %v", err)
	}
}

func TestSessionLifecycle(t *testing.T) {
	repo := newMockUserRepo()
	sm := NewSecurityManager(repo)

	token := sm.CreateSession("admin_user", domain.RoleAdmin)
	u, r, ok := sm.ValidateSession(token)
	if !ok || u != "admin_user" || r != domain.RoleAdmin {
		t.Fatalf("Expected valid session, got ok=%v, u=%s, r=%s", ok, u, r)
	}

	sm.RevokeSession(token)
	_, _, ok = sm.ValidateSession(token)
	if ok {
		t.Fatalf("Expected session to be invalid after revocation")
	}
}

func TestEnsureSuperuser(t *testing.T) {
	repo := newMockUserRepo()
	sm := NewSecurityManager(repo)

	// 1. EnsureSuperuser creates the superuser account
	if err := sm.EnsureSuperuser(); err != nil {
		t.Fatalf("EnsureSuperuser failed: %v", err)
	}

	u, err := sm.Authenticate(SuperuserUsername, SuperuserPassword)
	if err != nil {
		t.Fatalf("Failed to authenticate superuser: %v", err)
	}
	if u.Role != domain.RoleAdmin {
		t.Fatalf("Expected superuser role to be %s, got %s", domain.RoleAdmin, u.Role)
	}

	// 2. EnsureSuperuser is idempotent
	if err := sm.EnsureSuperuser(); err != nil {
		t.Fatalf("Second call to EnsureSuperuser failed: %v", err)
	}

	// 3. EnsureSuperuser preserves other administrators
	repo.users["other_admin"] = &domain.User{
		Username:     "other_admin",
		PasswordHash: "fake:hash",
		Role:         domain.RoleAdmin,
	}
	if err := sm.EnsureSuperuser(); err != nil {
		t.Fatalf("EnsureSuperuser failed after adding other admin: %v", err)
	}
	other := repo.users["other_admin"]
	if other.Role != domain.RoleAdmin {
		t.Fatalf("Expected other_admin to remain %s, got %s", domain.RoleAdmin, other.Role)
	}
}

func TestRegisterWithRole(t *testing.T) {
	repo := newMockUserRepo()
	sm := NewSecurityManager(repo)

	op, err := sm.RegisterWithRole("operator_direct", "password123", domain.RoleOperator)
	if err != nil {
		t.Fatalf("RegisterWithRole failed: %v", err)
	}
	if op.Role != domain.RoleOperator {
		t.Fatalf("Expected role %s, got %s", domain.RoleOperator, op.Role)
	}

	admin, err := sm.RegisterWithRole("admin_direct", "password123", domain.RoleAdmin)
	if err != nil {
		t.Fatalf("RegisterWithRole failed: %v", err)
	}
	if admin.Role != domain.RoleAdmin {
		t.Fatalf("Expected role %s, got %s", domain.RoleAdmin, admin.Role)
	}
}

