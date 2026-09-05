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
// File: internal/storage/user_repo_test.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package storage

import (
	"path/filepath"
	"testing"
	"time"

	"noxfort-monitor-server/internal/domain"
)

func TestUserRepositorySQLite(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_users.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	// Verify initial admin count is 0
	count, err := repo.CountAdmins()
	if err != nil {
		t.Fatalf("Failed to count admins: %v", err)
	}
	if count != 0 {
		t.Fatalf("Expected 0 admins initially, got %d", count)
	}

	// Create admin user
	adminUser := &domain.User{
		Username:     "prime_admin",
		PasswordHash: "fake:hash1",
		Role:         domain.RoleAdmin,
		CreatedAt:    time.Now(),
	}
	if err := repo.Create(adminUser); err != nil {
		t.Fatalf("Failed to create admin user: %v", err)
	}

	count, _ = repo.CountAdmins()
	if count != 1 {
		t.Fatalf("Expected 1 admin, got %d", count)
	}

	// Create operator user
	opUser := &domain.User{
		Username:     "plant_operator",
		PasswordHash: "fake:hash2",
		Role:         domain.RoleOperator,
		CreatedAt:    time.Now(),
	}
	if err := repo.Create(opUser); err != nil {
		t.Fatalf("Failed to create operator user: %v", err)
	}

	count, _ = repo.CountAdmins()
	if count != 1 {
		t.Fatalf("Admin count must still be 1 after operator created, got %d", count)
	}

	// List users
	users, err := repo.List()
	if err != nil {
		t.Fatalf("Failed to list users: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("Expected 2 users, got %d", len(users))
	}

	// Delete user by username
	if err := repo.DeleteByUsername("plant_operator"); err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	deleted, err := repo.GetByUsername("plant_operator")
	if err != nil || deleted != nil {
		t.Fatalf("Expected operator to be deleted, got %v", deleted)
	}
}
