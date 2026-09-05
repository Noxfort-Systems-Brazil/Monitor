// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: internal/storage/contact_repo.go
// Author: Gabriel Moraes
// Date: 2026-01-18
// Modified: 2026-09-04 (PostgreSQL & Hot-Reload Support)

package storage

import (
	"database/sql"
	"fmt"
	"sync"

	"noxfort-monitor-server/internal/domain"
)

// ContactRepositorySQLite implements domain.ContactRepository with multi-database support.
type ContactRepositorySQLite struct {
	mu     sync.RWMutex
	db     *sql.DB
	driver string
}

// NewContactRepository creates a new instance.
func NewContactRepository(db *sql.DB) *ContactRepositorySQLite {
	return &ContactRepositorySQLite{db: db, driver: "sqlite"}
}

// SetDB updates the database connection and dialect at runtime.
func (r *ContactRepositorySQLite) SetDB(db *sql.DB, driver string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.db = db
	r.driver = driver
}

func (r *ContactRepositorySQLite) getDB() (*sql.DB, string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.db, r.driver
}

// GetAllContacts returns all response team members.
func (r *ContactRepositorySQLite) GetAllContacts() ([]domain.Contact, error) {
	db, _ := r.getDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `SELECT id, name, email, phone, role, notify_critical, enabled, telegram_chat_id FROM contacts;`

	rows, err := db.Query(query)
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
	db, driver := r.getDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	query := AdaptQuery(`
		INSERT INTO contacts (name, email, phone, role, notify_critical, enabled, telegram_chat_id)
		VALUES (?, ?, ?, ?, ?, ?, ?);
	`, driver)

	_, err := db.Exec(query, c.Name, c.Email, c.Phone, c.Role, c.NotifyCritical, c.Enabled, c.TelegramChatID)
	if err != nil {
		return fmt.Errorf("failed to insert contact: %w", err)
	}
	return nil
}

// UpdateContact modifies an existing person on the team.
func (r *ContactRepositorySQLite) UpdateContact(c *domain.Contact) error {
	db, driver := r.getDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	query := AdaptQuery(`
		UPDATE contacts
		SET name = ?, email = ?, phone = ?, role = ?, notify_critical = ?, enabled = ?, telegram_chat_id = ?
		WHERE id = ?;
	`, driver)

	_, err := db.Exec(query, c.Name, c.Email, c.Phone, c.Role, c.NotifyCritical, c.Enabled, c.TelegramChatID, c.ID)
	if err != nil {
		return fmt.Errorf("failed to update contact: %w", err)
	}
	return nil
}

// DeleteContact removes a person by ID.
func (r *ContactRepositorySQLite) DeleteContact(id int64) error {
	db, driver := r.getDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	query := AdaptQuery(`DELETE FROM contacts WHERE id = ?;`, driver)

	_, err := db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete contact: %w", err)
	}
	return nil
}
