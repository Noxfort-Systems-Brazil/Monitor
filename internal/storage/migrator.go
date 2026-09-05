// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: internal/storage/migrator.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package storage

import (
	"database/sql"
	"fmt"
	"log"
)

// MigrateData copies devices, contacts, settings, and users from sourceDB to targetDB.
func MigrateData(sourceDB *sql.DB, sourceDriver string, targetDB *sql.DB, targetDriver string) error {
	if sourceDB == nil || targetDB == nil {
		return fmt.Errorf("source and target databases must be initialized")
	}

	log.Printf("[MIGRATOR] Starting data synchronization from %s to %s...", sourceDriver, targetDriver)

	// 1. Migrate Devices
	devRows, err := sourceDB.Query("SELECT name, identifier, last_seen, enabled FROM devices")
	if err == nil {
		defer devRows.Close()
		insertDevQuery := AdaptQuery(
			"INSERT INTO devices (name, identifier, last_seen, enabled) VALUES (?, ?, ?, ?) ON CONFLICT (identifier) DO NOTHING;",
			targetDriver,
		)
		for devRows.Next() {
			var name, identifier string
			var lastSeen sql.NullTime
			var enabled bool
			if err := devRows.Scan(&name, &identifier, &lastSeen, &enabled); err == nil {
				_, _ = targetDB.Exec(insertDevQuery, name, identifier, lastSeen, enabled)
			}
		}
	}

	// 2. Migrate Contacts
	contactRows, err := sourceDB.Query("SELECT name, email, phone, role, notify_critical, enabled, telegram_chat_id FROM contacts")
	if err == nil {
		defer contactRows.Close()
		insertContactQuery := AdaptQuery(
			"INSERT INTO contacts (name, email, phone, role, notify_critical, enabled, telegram_chat_id) VALUES (?, ?, ?, ?, ?, ?, ?);",
			targetDriver,
		)
		for contactRows.Next() {
			var name, email, phone, role, tg string
			var crit, en bool
			if err := contactRows.Scan(&name, &email, &phone, &role, &crit, &en, &tg); err == nil {
				// Check if email already exists in target to avoid duplicates
				var exists bool
				checkQuery := AdaptQuery("SELECT EXISTS(SELECT 1 FROM contacts WHERE email = ?);", targetDriver)
				_ = targetDB.QueryRow(checkQuery, email).Scan(&exists)
				if !exists {
					_, _ = targetDB.Exec(insertContactQuery, name, email, phone, role, crit, en, tg)
				}
			}
		}
	}

	// 3. Migrate Users
	userRows, err := sourceDB.Query("SELECT username, password_hash, role, created_at FROM users")
	if err == nil {
		defer userRows.Close()
		insertUserQuery := AdaptQuery(
			"INSERT INTO users (username, password_hash, role, created_at) VALUES (?, ?, ?, ?) ON CONFLICT (username) DO NOTHING;",
			targetDriver,
		)
		for userRows.Next() {
			var u, p, r string
			var t sql.NullTime
			if err := userRows.Scan(&u, &p, &r, &t); err == nil {
				_, _ = targetDB.Exec(insertUserQuery, u, p, r, t)
			}
		}
	}

	// 4. Migrate Settings (copy active settings if target is still default)
	var s struct {
		smtpHost, smtpUser, smtpPass, smtpFrom, adminEmail, mqtt, tg, ngTok, ngDom string
		smtpPort                                                                    int
		enabled, ngEn                                                               bool
	}
	settingsRow := sourceDB.QueryRow(`
		SELECT smtp_host, smtp_port, smtp_user, smtp_pass, smtp_from, admin_email,
		       mqtt_address, enabled, telegram_bot_token,
		       COALESCE(ngrok_auth_token, ''), COALESCE(ngrok_domain, ''), COALESCE(ngrok_enabled, 0)
		FROM settings WHERE id = 1
	`)
	if err := settingsRow.Scan(
		&s.smtpHost, &s.smtpPort, &s.smtpUser, &s.smtpPass, &s.smtpFrom, &s.adminEmail,
		&s.mqtt, &s.enabled, &s.tg, &s.ngTok, &s.ngDom, &s.ngEn,
	); err == nil {
		updateSettingsQuery := AdaptQuery(`
			UPDATE settings SET
				smtp_host = ?, smtp_port = ?, smtp_user = ?, smtp_pass = ?,
				smtp_from = ?, admin_email = ?, mqtt_address = ?, enabled = ?,
				telegram_bot_token = ?, ngrok_auth_token = ?, ngrok_domain = ?, ngrok_enabled = ?
			WHERE id = 1;
		`, targetDriver)
		_, _ = targetDB.Exec(updateSettingsQuery,
			s.smtpHost, s.smtpPort, s.smtpUser, s.smtpPass,
			s.smtpFrom, s.adminEmail, s.mqtt, s.enabled,
			s.tg, s.ngTok, s.ngDom, s.ngEn,
		)
	}

	log.Printf("[MIGRATOR] Data synchronization completed successfully.")
	return nil
}
