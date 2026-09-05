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
// File: internal/storage/contact_repo_test.go
// Author: Gabriel Moraes
// Date: 2026-01-20

package storage

import (
	"os"
	"path/filepath"
	"testing"

	"noxfort-monitor-server/internal/domain"
)

func TestContactRepository_CRUD(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_contacts.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	defer os.Remove(dbPath)

	repo := NewContactRepository(db)

	// 1. Create Contact
	contact := &domain.Contact{
		Name:           "Alice Smith",
		Email:          "alice@noxfort.com",
		Phone:          "+55 11 99999-0001",
		Role:           "System Admin",
		NotifyCritical: false,
		Enabled:        true,
		TelegramChatID: "12345678",
	}

	if err := repo.CreateContact(contact); err != nil {
		t.Fatalf("Failed to create contact: %v", err)
	}

	// 2. Get All Contacts
	contacts, err := repo.GetAllContacts()
	if err != nil {
		t.Fatalf("Failed to get contacts: %v", err)
	}
	if len(contacts) != 1 {
		t.Fatalf("Expected 1 contact, got %d", len(contacts))
	}
	saved := contacts[0]
	if saved.Name != "Alice Smith" || saved.Email != "alice@noxfort.com" || saved.TelegramChatID != "12345678" {
		t.Errorf("Unexpected contact data: %+v", saved)
	}

	// 3. Update Contact
	saved.Name = "Alice S. Modified"
	saved.Role = "Technician"
	saved.Email = "alice.mod@noxfort.com"
	saved.Phone = "+55 11 98888-0002"
	saved.NotifyCritical = true
	saved.TelegramChatID = "87654321"

	if err := repo.UpdateContact(&saved); err != nil {
		t.Fatalf("Failed to update contact: %v", err)
	}

	// Verify Update
	contactsAfterUpdate, err := repo.GetAllContacts()
	if err != nil {
		t.Fatalf("Failed to get contacts after update: %v", err)
	}
	if len(contactsAfterUpdate) != 1 {
		t.Fatalf("Expected 1 contact, got %d", len(contactsAfterUpdate))
	}
	updated := contactsAfterUpdate[0]
	if updated.Name != "Alice S. Modified" ||
		updated.Role != "Technician" ||
		updated.Email != "alice.mod@noxfort.com" ||
		updated.Phone != "+55 11 98888-0002" ||
		!updated.NotifyCritical ||
		updated.TelegramChatID != "87654321" {
		t.Errorf("Updated contact mismatch: %+v", updated)
	}

	// 4. Delete Contact
	if err := repo.DeleteContact(updated.ID); err != nil {
		t.Fatalf("Failed to delete contact: %v", err)
	}

	contactsAfterDelete, err := repo.GetAllContacts()
	if err != nil {
		t.Fatalf("Failed to get contacts after delete: %v", err)
	}
	if len(contactsAfterDelete) != 0 {
		t.Fatalf("Expected 0 contacts after delete, got %d", len(contactsAfterDelete))
	}
}
