// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: internal/storage/postgres_schema.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package storage

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

// CheckPostgresSchemaExists inspects information_schema to verify whether the specified schema exists.
func CheckPostgresSchemaExists(db *sql.DB, schemaName string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = $1);`
	var exists bool
	err := db.QueryRow(query, schemaName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check schema existence: %w", err)
	}
	return exists, nil
}

// InitPostgresSchema idempotently creates the schema and all tables, constraints, and audit indexes.
func InitPostgresSchema(db *sql.DB, schemaName string) error {
	if schemaName == "" {
		schemaName = "schema_monitor"
	}

	// 1. Create Schema if not exists (Sanitize schema identifier to prevent injection)
	sanitizedSchema := strings.ReplaceAll(schemaName, "\"", "")
	createSchemaSQL := fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS "%s";`, sanitizedSchema)
	if _, err := db.Exec(createSchemaSQL); err != nil {
		return fmt.Errorf("failed to create schema '%s': %w", sanitizedSchema, err)
	}

	// Set search path for current session
	setSearchPathSQL := fmt.Sprintf(`SET search_path TO "%s", public;`, sanitizedSchema)
	if _, err := db.Exec(setSearchPathSQL); err != nil {
		return fmt.Errorf("failed to set search_path: %w", err)
	}

	// 2. DDL statements for PostgreSQL tables and audit logs
	statements := []string{
		// Devices
		`CREATE TABLE IF NOT EXISTS devices (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			identifier TEXT UNIQUE NOT NULL,
			last_seen TIMESTAMPTZ,
			enabled BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		);`,

		// Contacts
		`CREATE TABLE IF NOT EXISTS contacts (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL,
			phone TEXT,
			role TEXT,
			notify_critical BOOLEAN DEFAULT TRUE,
			enabled BOOLEAN DEFAULT TRUE,
			telegram_chat_id TEXT DEFAULT ''
		);`,

		// Settings
		`CREATE TABLE IF NOT EXISTS settings (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			smtp_host TEXT DEFAULT '',
			smtp_port INTEGER DEFAULT 587,
			smtp_user TEXT DEFAULT '',
			smtp_pass TEXT DEFAULT '',
			smtp_from TEXT DEFAULT '',
			admin_email TEXT DEFAULT '',
			mqtt_address TEXT DEFAULT 'tcp://127.0.0.1:1883',
			telegram_bot_token TEXT DEFAULT '',
			ngrok_auth_token TEXT DEFAULT '',
			ngrok_domain TEXT DEFAULT '',
			ngrok_enabled BOOLEAN DEFAULT FALSE,
			enabled BOOLEAN DEFAULT FALSE
		);`,

		// Ensure default settings row
		`INSERT INTO settings (id) VALUES (1) ON CONFLICT (id) DO NOTHING;`,

		// Telemetry / Incidents
		`CREATE TABLE IF NOT EXISTS telemetry (
			id BIGSERIAL PRIMARY KEY,
			category TEXT,
			device_id TEXT NOT NULL,
			origin TEXT NOT NULL,
			level TEXT NOT NULL,
			message TEXT,
			occurred_at TIMESTAMPTZ,
			received_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		);`,

		// Users
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'OPERATOR',
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		);`,

		// 3. AUDIT TABLES
		// Security & Access Control Audit Logs (Mirrors Synapse security_audit_logs)
		`CREATE TABLE IF NOT EXISTS security_audit_logs (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
			username VARCHAR(128) NOT NULL,
			action VARCHAR(128) NOT NULL,
			details TEXT,
			ip_address VARCHAR(64)
		);`,

		// Alert Notification Dispatch Logs (SLA & Delivery verification)
		`CREATE TABLE IF NOT EXISTS alert_dispatch_logs (
			id BIGSERIAL PRIMARY KEY,
			telemetry_id BIGINT,
			channel VARCHAR(32) NOT NULL,
			recipient VARCHAR(255) NOT NULL,
			role VARCHAR(64),
			status VARCHAR(32) NOT NULL,
			error_reason TEXT,
			dispatched_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		);`,

		// Watchdog Availability & State Transitions (Uptime & MTTR Tracking)
		`CREATE TABLE IF NOT EXISTS device_state_transitions (
			id BIGSERIAL PRIMARY KEY,
			device_identifier VARCHAR(128) NOT NULL,
			previous_state VARCHAR(32),
			new_state VARCHAR(32) NOT NULL,
			duration_offline_sec BIGINT DEFAULT 0,
			transition_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		);`,

		// 4. PERFORMANCE & COMPOSITE INDEXES
		`CREATE INDEX IF NOT EXISTS idx_telemetry_occurred ON telemetry (occurred_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_telemetry_device_id ON telemetry (device_id);`,
		`CREATE INDEX IF NOT EXISTS idx_telemetry_category ON telemetry (category);`,
		`CREATE INDEX IF NOT EXISTS idx_telemetry_device_occurred ON telemetry (device_id, occurred_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_telemetry_level_occurred ON telemetry (level, occurred_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_sec_audit_created ON security_audit_logs (created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_sec_audit_user ON security_audit_logs (username);`,
		`CREATE INDEX IF NOT EXISTS idx_alert_dispatch_time ON alert_dispatch_logs (dispatched_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_alert_dispatch_telemetry ON alert_dispatch_logs (telemetry_id);`,
		`CREATE INDEX IF NOT EXISTS idx_device_trans_time ON device_state_transitions (device_identifier, transition_at DESC);`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("failed executing DDL statement in schema '%s': %w", sanitizedSchema, err)
		}
	}

	log.Printf("[POSTGRES] Schema '%s' and tables initialized successfully.", sanitizedSchema)
	return nil
}
