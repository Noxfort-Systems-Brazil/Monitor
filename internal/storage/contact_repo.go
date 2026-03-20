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
// File: internal/storage/contact_repo.go
// Author: Gabriel Moraes
// Date: 2026-01-18

package storage

import (
	"database/sql"
	"fmt"

	"noxfort-monitor-server/internal/domain"
)

// ContactRepositorySQLite implements domain.ContactRepository.
type ContactRepositorySQLite struct {
	db *sql.DB
}

// NewContactRepository creates a new instance.
func NewContactRepository(db *sql.DB) *ContactRepositorySQLite {
	return &ContactRepositorySQLite{db: db}
}

// GetAllContacts returns all response team members.
func (r *ContactRepositorySQLite) GetAllContacts() ([]domain.Contact, error) {
	query := `SELECT id, name, email, phone, role, notify_critical, enabled, telegram_chat_id FROM contacts`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query contacts: %w", err)
	}
	defer rows.Close()

	var contacts []domain.Contact
	for rows.Next() {
		var c domain.Contact
		err := rows.Scan(
			&c.ID,
			&c.Name,
			&c.Email,
			&c.Phone,
			&c.Role,
			&c.NotifyCritical,
			&c.Enabled,
			&c.TelegramChatID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan contact: %w", err)
		}
		contacts = append(contacts, c)
	}

	return contacts, nil
}

// CreateContact adds a new person to the team.
func (r *ContactRepositorySQLite) CreateContact(c *domain.Contact) error {
	query := `
	INSERT INTO contacts (name, email, phone, role, notify_critical, enabled, telegram_chat_id)
	VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.Exec(query, c.Name, c.Email, c.Phone, c.Role, c.NotifyCritical, c.Enabled, c.TelegramChatID)
	if err != nil {
		return fmt.Errorf("failed to insert contact: %w", err)
	}
	return nil
}

// DeleteContact removes a person by ID.
func (r *ContactRepositorySQLite) DeleteContact(id int64) error {
	query := `DELETE FROM contacts WHERE id = ?`

	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete contact: %w", err)
	}
	return nil
}
