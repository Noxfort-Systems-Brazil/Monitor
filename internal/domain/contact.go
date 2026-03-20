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
// File: internal/domain/contact.go
// Author: Gabriel Moraes
// Date: 2026-01-18

package domain

// Contact represents a member of the response team or an external notification endpoint.
// It maps to the 'contacts' table in the database.
type Contact struct {
	// ID is the unique identifier for the contact (Auto-increment).
	ID int64 `json:"id" db:"id"`

	// Name of the person or team (e.g., "Maintenance Team A").
	Name string `json:"name" db:"name"`

	// Role helps categorize the contact (e.g., "SysAdmin", "Manager").
	// Matches the 'role' column in the database.
	Role string `json:"role" db:"role"`

	// Email is the primary notification channel.
	Email string `json:"email" db:"email"`

	// Phone is used for SMS or urgent calls (optional).
	Phone string `json:"phone" db:"phone"`

	// NotifyCritical determines if this contact receives high-priority alerts.
	// Matches the 'notify_critical' column in the database.
	NotifyCritical bool `json:"notify_critical" db:"notify_critical"`

	// Enabled allows temporarily disabling a contact without deleting the record.
	Enabled bool `json:"enabled" db:"enabled"`

	// TelegramChatID is the personal Telegram Chat ID for sending direct alerts.
	// Can be obtained by messaging @userinfobot on Telegram.
	TelegramChatID string `json:"telegram_chat_id" db:"telegram_chat_id"`
}

// ContactRepository defines the contract for managing contact data.
// This interface decouples the HTTP handlers and AlertService from the SQL implementation.
type ContactRepository interface {
	// GetAllContacts retrieves the full list of registered contacts.
	GetAllContacts() ([]Contact, error)

	// CreateContact adds a new contact to the database.
	CreateContact(contact *Contact) error

	// DeleteContact removes a contact by its ID.
	DeleteContact(id int64) error
}
