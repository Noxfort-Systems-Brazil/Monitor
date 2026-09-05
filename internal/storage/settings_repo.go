// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: internal/storage/settings_repo.go
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

// SettingsRepositorySQLite implements domain.SettingsRepository with multi-database support.
type SettingsRepositorySQLite struct {
	mu     sync.RWMutex
	db     *sql.DB
	driver string
}

// NewSettingsRepository creates a new instance.
func NewSettingsRepository(db *sql.DB) *SettingsRepositorySQLite {
	return &SettingsRepositorySQLite{db: db, driver: "sqlite"}
}

// SetDB updates the database connection and dialect at runtime.
func (r *SettingsRepositorySQLite) SetDB(db *sql.DB, driver string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.db = db
	r.driver = driver
}

func (r *SettingsRepositorySQLite) getDB() (*sql.DB, string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.db, r.driver
}

// GetSMTPSettings retrieves only the email configuration (used by AlertService).
func (r *SettingsRepositorySQLite) GetSMTPSettings() (*domain.SMTPSettings, error) {
	db, _ := r.getDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
	SELECT 
		smtp_host, smtp_port, smtp_user, smtp_pass, 
		smtp_from, admin_email, enabled
	FROM settings 
	WHERE id = 1;`

	var s domain.SMTPSettings
	err := db.QueryRow(query).Scan(
		&s.Host,
		&s.Port,
		&s.Username,
		&s.Password,
		&s.FromEmail,
		&s.AdminEmail,
		&s.Enabled,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return &domain.SMTPSettings{Port: 587}, nil
		}
		return nil, fmt.Errorf("failed to fetch smtp settings: %w", err)
	}

	return &s, nil
}

// SaveSMTPSettings updates only the email configuration.
func (r *SettingsRepositorySQLite) SaveSMTPSettings(s *domain.SMTPSettings) error {
	db, driver := r.getDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	query := AdaptQuery(`
	UPDATE settings SET
		smtp_host = ?,
		smtp_port = ?,
		smtp_user = ?,
		smtp_pass = ?,
		smtp_from = ?,
		admin_email = ?,
		enabled = ?
	WHERE id = 1;`, driver)

	_, err := db.Exec(query,
		s.Host,
		s.Port,
		s.Username,
		s.Password,
		s.FromEmail,
		s.AdminEmail,
		s.Enabled,
	)

	if err != nil {
		return fmt.Errorf("failed to save smtp settings: %w", err)
	}
	return nil
}

// GetSettings retrieves the full system configuration.
func (r *SettingsRepositorySQLite) GetSettings() (*domain.Settings, error) {
	db, _ := r.getDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
	SELECT 
		id, smtp_host, smtp_port, smtp_user, smtp_pass, 
		smtp_from, admin_email, mqtt_address, enabled, telegram_bot_token,
		COALESCE(ngrok_auth_token, ''), COALESCE(ngrok_domain, ''), COALESCE(ngrok_enabled, false)
	FROM settings 
	WHERE id = 1;`

	var s domain.Settings
	err := db.QueryRow(query).Scan(
		&s.ID,
		&s.SMTPHost,
		&s.SMTPPort,
		&s.SMTPUser,
		&s.SMTPPass,
		&s.SMTPFrom,
		&s.AdminEmail,
		&s.MqttAddress,
		&s.Enabled,
		&s.TelegramBotToken,
		&s.NgrokAuthToken,
		&s.NgrokDomain,
		&s.NgrokEnabled,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return &domain.Settings{
				SMTPPort:    587,
				MqttAddress: "tcp://127.0.0.1:1883",
			}, nil
		}
		return nil, fmt.Errorf("failed to fetch full settings: %w", err)
	}

	return &s, nil
}

// SaveSettings updates the full system configuration.
func (r *SettingsRepositorySQLite) SaveSettings(s *domain.Settings) error {
	db, driver := r.getDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	query := AdaptQuery(`
	UPDATE settings SET
		smtp_host = ?,
		smtp_port = ?,
		smtp_user = ?,
		smtp_pass = ?,
		smtp_from = ?,
		admin_email = ?,
		mqtt_address = ?,
		enabled = ?,
		telegram_bot_token = ?,
		ngrok_auth_token = ?,
		ngrok_domain = ?,
		ngrok_enabled = ?
	WHERE id = 1;`, driver)

	_, err := db.Exec(query,
		s.SMTPHost,
		s.SMTPPort,
		s.SMTPUser,
		s.SMTPPass,
		s.SMTPFrom,
		s.AdminEmail,
		s.MqttAddress,
		s.Enabled,
		s.TelegramBotToken,
		s.NgrokAuthToken,
		s.NgrokDomain,
		s.NgrokEnabled,
	)

	if err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}
	return nil
}
