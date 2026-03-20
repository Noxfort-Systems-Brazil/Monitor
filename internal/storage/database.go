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
// File: internal/storage/database.go
// Author: Gabriel Moraes
// Date: 2026-01-19

package storage

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "modernc.org/sqlite" // Pure Go SQLite Driver
)

// NewDatabase opens a connection to the SQLite database and initializes the schema.
func NewDatabase(path string) (*sql.DB, error) {
	// Using pure Go driver
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Enable Foreign Keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Initialize Tables (Schema Migration)
	if err := initSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

// initSchema creates the necessary tables if they don't exist.
func initSchema(db *sql.DB) error {
	// 1. Devices Table (Simplified V2)
	// Now tracks 'identifiers' (origins) instead of hardware IDs.
	queryDevices := `
	CREATE TABLE IF NOT EXISTS devices (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		identifier TEXT UNIQUE NOT NULL, -- Matches JSON "origin" (e.g. "synapse")
		last_seen DATETIME,
		enabled BOOLEAN DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := db.Exec(queryDevices); err != nil {
		return fmt.Errorf("error creating devices table: %w", err)
	}

	// 2. Contacts Table
	// Roles: "system_admin", "technician", "programmer"
	queryContacts := `
	CREATE TABLE IF NOT EXISTS contacts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT NOT NULL,
		phone TEXT,
		role TEXT, 
		notify_critical BOOLEAN DEFAULT 1,
		enabled BOOLEAN DEFAULT 1,
		telegram_chat_id TEXT DEFAULT ''
	);`
	if _, err := db.Exec(queryContacts); err != nil {
		return fmt.Errorf("error creating contacts table: %w", err)
	}

	// 2.1 Migration: Ensure telegram_chat_id exists for existing databases.
	migrationContactTelegram := "ALTER TABLE contacts ADD COLUMN telegram_chat_id TEXT DEFAULT '';"
	if _, err := db.Exec(migrationContactTelegram); err != nil {
		if !strings.Contains(err.Error(), "duplicate column") {
			log.Printf("[STORAGE] Migration Note: %v (Normal if column exists)", err)
		}
	}

	// 3. Settings Table
	querySettings := `
	CREATE TABLE IF NOT EXISTS settings (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		smtp_host TEXT DEFAULT '',
		smtp_port INTEGER DEFAULT 587,
		smtp_user TEXT DEFAULT '',
		smtp_pass TEXT DEFAULT '',
		smtp_from TEXT DEFAULT '',
		admin_email TEXT DEFAULT '',
		mqtt_address TEXT DEFAULT 'tcp://127.0.0.1:1883',
		enabled BOOLEAN DEFAULT 0
	);`
	if _, err := db.Exec(querySettings); err != nil {
		return fmt.Errorf("error creating settings table: %w", err)
	}

	// 3.1 Migration: Ensure mqtt_address exists for updated databases.
	migrationMqtt := "ALTER TABLE settings ADD COLUMN mqtt_address TEXT DEFAULT 'tcp://127.0.0.1:1883';"
	if _, err := db.Exec(migrationMqtt); err != nil {
		if !strings.Contains(err.Error(), "duplicate column") {
			log.Printf("[STORAGE] Migration Note: %v (This is normal if column exists)", err)
		}
	}

	// 3.2 Migration: Ensure telegram_bot_token exists.
	migrationTelegramToken := "ALTER TABLE settings ADD COLUMN telegram_bot_token TEXT DEFAULT '';"
	if _, err := db.Exec(migrationTelegramToken); err != nil {
		if !strings.Contains(err.Error(), "duplicate column") {
			log.Printf("[STORAGE] Migration Note: %v (Normal if column exists)", err)
		}
	}

	// Ensure the default settings row exists
	queryInitSettings := `INSERT OR IGNORE INTO settings (id) VALUES (1);`
	if _, err := db.Exec(queryInitSettings); err != nil {
		return fmt.Errorf("error initializing settings row: %w", err)
	}

	// 4. Telemetry Table (Refined V2)
	// Updated to include 'category' for routing (HARDWARE/SOFTWARE)
	queryTelemetry := `
	CREATE TABLE IF NOT EXISTS telemetry (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		category TEXT,            -- NEW: "HARDWARE" or "SOFTWARE"
		device_id TEXT NOT NULL,  -- Stores the "identifier" (origin)
		origin TEXT NOT NULL,     -- Redundant but kept for query speed
		level TEXT NOT NULL,      -- "INFO", "CRITICAL", etc.
		message TEXT,             -- Human readable alert text
		occurred_at DATETIME,     -- ISO 8601 Timestamp from source
		received_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(queryTelemetry); err != nil {
		return fmt.Errorf("error creating telemetry table: %w", err)
	}

	// 4.1 Migration: Ensure category column exists
	migrationCategory := "ALTER TABLE telemetry ADD COLUMN category TEXT;"
	if _, err := db.Exec(migrationCategory); err != nil {
		if !strings.Contains(err.Error(), "duplicate column") {
			log.Printf("[STORAGE] Migration Note: %v (Normal if category exists)", err)
		}
	}

	log.Println("[STORAGE] Database schema initialized successfully.")
	return nil
}
