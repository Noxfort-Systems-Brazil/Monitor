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
// File: internal/transport/http/contact_handler_test.go
// Author: Gabriel Moraes
// Date: 2026-01-20

package http

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"noxfort-monitor-server/internal/domain"
)

type mockContactRepo struct {
	contacts []domain.Contact
	nextID   int64
}

func (m *mockContactRepo) GetAllContacts() ([]domain.Contact, error) {
	return m.contacts, nil
}

func (m *mockContactRepo) CreateContact(c *domain.Contact) error {
	m.nextID++
	c.ID = m.nextID
	m.contacts = append(m.contacts, *c)
	return nil
}

func (m *mockContactRepo) UpdateContact(c *domain.Contact) error {
	for i, existing := range m.contacts {
		if existing.ID == c.ID {
			m.contacts[i] = *c
			return nil
		}
	}
	return nil
}

func (m *mockContactRepo) DeleteContact(id int64) error {
	for i, existing := range m.contacts {
		if existing.ID == id {
			m.contacts = append(m.contacts[:i], m.contacts[i+1:]...)
			return nil
		}
	}
	return nil
}

func TestContactHandler_CreateUpdateDelete(t *testing.T) {
	repo := &mockContactRepo{}
	handler := NewContactHandler(repo)

	// 1. Create
	createForm := url.Values{
		"name":             {"Bob Builder"},
		"role":             {"Programmer"},
		"email":            {"bob@noxfort.com"},
		"phone":            {"+55 11 91111-2222"},
		"telegram_chat_id": {"999888"},
		"notify_critical":  {"on"},
	}

	reqCreate := httptest.NewRequest(http.MethodPost, "/contacts/create", strings.NewReader(createForm.Encode()))
	reqCreate.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recCreate := httptest.NewRecorder()

	handler.HandleCreate(recCreate, reqCreate)

	if recCreate.Code != http.StatusSeeOther {
		t.Fatalf("Expected status 303, got %d", recCreate.Code)
	}

	if len(repo.contacts) != 1 {
		t.Fatalf("Expected 1 contact in repo, got %d", len(repo.contacts))
	}
	if repo.contacts[0].Name != "Bob Builder" || !repo.contacts[0].NotifyCritical {
		t.Errorf("Unexpected contact: %+v", repo.contacts[0])
	}

	// 2. Update
	updateForm := url.Values{
		"id":               {"1"},
		"name":             {"Bob The Builder"},
		"role":             {"System Admin"},
		"email":            {"bob.builder@noxfort.com"},
		"phone":            {"+55 11 93333-4444"},
		"telegram_chat_id": {"112233"},
		"notify_critical":  {""},
	}

	reqUpdate := httptest.NewRequest(http.MethodPost, "/contacts/update", strings.NewReader(updateForm.Encode()))
	reqUpdate.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recUpdate := httptest.NewRecorder()

	handler.HandleUpdate(recUpdate, reqUpdate)

	if recUpdate.Code != http.StatusSeeOther {
		t.Fatalf("Expected status 303, got %d", recUpdate.Code)
	}

	if len(repo.contacts) != 1 {
		t.Fatalf("Expected 1 contact in repo, got %d", len(repo.contacts))
	}
	updated := repo.contacts[0]
	if updated.Name != "Bob The Builder" ||
		updated.Role != "System Admin" ||
		updated.Email != "bob.builder@noxfort.com" ||
		updated.Phone != "+55 11 93333-4444" ||
		updated.TelegramChatID != "112233" ||
		updated.NotifyCritical != false {
		t.Errorf("Unexpected updated contact: %+v", updated)
	}

	// 3. Delete
	deleteForm := url.Values{
		"id": {"1"},
	}

	reqDelete := httptest.NewRequest(http.MethodPost, "/contacts/delete", strings.NewReader(deleteForm.Encode()))
	reqDelete.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recDelete := httptest.NewRecorder()

	handler.HandleDelete(recDelete, reqDelete)

	if recDelete.Code != http.StatusSeeOther {
		t.Fatalf("Expected status 303, got %d", recDelete.Code)
	}

	if len(repo.contacts) != 0 {
		t.Fatalf("Expected 0 contacts after delete, got %d", len(repo.contacts))
	}
}
