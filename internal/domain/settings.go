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
// File: internal/domain/settings.go
// Author: Gabriel Moraes
// Date: 2026-01-18

package domain

// Settings represents the full system configuration stored in the database.
// It maps directly to the columns in the 'settings' table.
type Settings struct {
	ID               int    `json:"id"`
	SMTPHost         string `json:"smtp_host"`
	SMTPPort         int    `json:"smtp_port"`
	SMTPUser         string `json:"smtp_user"`
	SMTPPass         string `json:"smtp_pass"`
	SMTPFrom         string `json:"smtp_from"`
	AdminEmail       string `json:"admin_email"`
	MqttAddress      string `json:"mqtt_address"`       // Connection string for devices (e.g., tcp://127.0.0.1:1883)
	Enabled          bool   `json:"enabled"`            // Master switch for email notifications
	TelegramBotToken string `json:"telegram_bot_token"` // Telegram Bot API Token
}

// SMTPSettings is a subset of settings specifically for the AlertService.
// This ensures the AlertService doesn't need to know about UI-specific fields like MqttAddress.
type SMTPSettings struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	FromEmail  string `json:"from_email"`
	AdminEmail string `json:"admin_email"`
	Enabled    bool   `json:"enabled"`
}

// SettingsRepository defines the contract for managing system configurations.
type SettingsRepository interface {
	// Methods for the AlertService (SMTP only)
	GetSMTPSettings() (*SMTPSettings, error)
	SaveSMTPSettings(settings *SMTPSettings) error

	// Methods for the Settings UI and Device Dashboard (Full Config)
	GetSettings() (*Settings, error)
	SaveSettings(settings *Settings) error
}
