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
// File: internal/storage/settings_repo.go
// Author: Gabriel Moraes
// Date: 2026-01-18

package storage

import (
	"database/sql"
	"fmt"

	"noxfort-monitor-server/internal/domain"
)

// SettingsRepositorySQLite implements domain.SettingsRepository.
type SettingsRepositorySQLite struct {
	db *sql.DB
}

// NewSettingsRepository creates a new instance.
func NewSettingsRepository(db *sql.DB) *SettingsRepositorySQLite {
	return &SettingsRepositorySQLite{db: db}
}

// GetSMTPSettings retrieves only the email configuration (used by AlertService).
func (r *SettingsRepositorySQLite) GetSMTPSettings() (*domain.SMTPSettings, error) {
	// Note: 'smtp_from' and 'enabled' columns must exist in the DB schema.
	query := `
	SELECT 
		smtp_host, smtp_port, smtp_user, smtp_pass, 
		smtp_from, admin_email, enabled
	FROM settings 
	WHERE id = 1`

	var s domain.SMTPSettings
	// Map SQL columns to Struct fields
	err := r.db.QueryRow(query).Scan(
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
			// Return default/empty settings instead of error for UX
			return &domain.SMTPSettings{Port: 587}, nil
		}
		return nil, fmt.Errorf("failed to fetch smtp settings: %w", err)
	}

	return &s, nil
}

// SaveSMTPSettings updates only the email configuration.
// Kept for backward compatibility if needed, though mostly superseded by SaveSettings.
func (r *SettingsRepositorySQLite) SaveSMTPSettings(s *domain.SMTPSettings) error {
	query := `
	UPDATE settings SET
		smtp_host = ?,
		smtp_port = ?,
		smtp_user = ?,
		smtp_pass = ?,
		smtp_from = ?,
		admin_email = ?,
		enabled = ?
	WHERE id = 1`

	_, err := r.db.Exec(query,
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

// GetSettings retrieves the full system configuration, including MQTT Address.
// Used by the Settings Handler to populate the UI.
func (r *SettingsRepositorySQLite) GetSettings() (*domain.Settings, error) {
	query := `
	SELECT 
		id, smtp_host, smtp_port, smtp_user, smtp_pass, 
		smtp_from, admin_email, mqtt_address, enabled, telegram_bot_token
	FROM settings 
	WHERE id = 1`

	var s domain.Settings
	err := r.db.QueryRow(query).Scan(
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
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Default configuration if DB is empty
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
	query := `
	UPDATE settings SET
		smtp_host = ?,
		smtp_port = ?,
		smtp_user = ?,
		smtp_pass = ?,
		smtp_from = ?,
		admin_email = ?,
		mqtt_address = ?,
		enabled = ?,
		telegram_bot_token = ?
	WHERE id = 1`

	_, err := r.db.Exec(query,
		s.SMTPHost,
		s.SMTPPort,
		s.SMTPUser,
		s.SMTPPass,
		s.SMTPFrom,
		s.AdminEmail,
		s.MqttAddress,
		s.Enabled,
		s.TelegramBotToken,
	)

	if err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}
	return nil
}
