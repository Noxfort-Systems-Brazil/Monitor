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
// File: internal/transport/http/settings_handler.go
// Author: Gabriel Moraes
// Date: 2026-01-18
// Modified: 2026-09-04 (SOLID Refactor)

package http

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"noxfort-monitor-server/internal/appdir"
	"noxfort-monitor-server/internal/domain"
)

// ConnectionTester defines the contract required to test notification channel credentials.
type ConnectionTester interface {
	TestConnection(settings *domain.Settings, to string) error
	TestTelegramConnection(botToken, chatID string) error
}

// SettingsHandler manages the configuration page and actions.
type SettingsHandler struct {
	repo   domain.SettingsRepository
	tester ConnectionTester
}

// NewSettingsHandler creates a handler for system settings with an injected ConnectionTester.
func NewSettingsHandler(r domain.SettingsRepository, t ConnectionTester) *SettingsHandler {
	return &SettingsHandler{
		repo:   r,
		tester: t,
	}
}

// parseSettingsForm extracts, sanitizes, and binds form parameters into a domain.Settings instance.
func parseSettingsForm(r *http.Request) *domain.Settings {
	port, _ := strconv.Atoi(r.FormValue("smtp_port"))
	smtpUser := strings.TrimSpace(r.FormValue("smtp_user"))

	adminEmail := strings.TrimSpace(r.FormValue("admin_email"))
	if adminEmail == "" {
		adminEmail = smtpUser
	}

	return &domain.Settings{
		SMTPHost:         strings.TrimSpace(r.FormValue("smtp_host")),
		SMTPPort:         port,
		SMTPUser:         smtpUser,
		SMTPPass:         r.FormValue("smtp_pass"),
		SMTPFrom:         strings.TrimSpace(r.FormValue("smtp_from")),
		AdminEmail:       adminEmail,
		MqttAddress:      strings.TrimSpace(r.FormValue("mqtt_address")),
		Enabled:          true,
		TelegramBotToken: strings.TrimSpace(r.FormValue("telegram_bot_token")),
	}
}

// ServePage renders the settings form with current values.
func (h *SettingsHandler) ServePage(w http.ResponseWriter, r *http.Request) {
	settings, err := h.repo.GetSettings()
	if err != nil {
		log.Printf("[SETTINGS] Failed to load settings: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles(
		appdir.Path("web/templates/layout.html"),
		appdir.Path("web/templates/settings.html"),
		appdir.Path("web/templates/partials/settings_smtp.html"),
		appdir.Path("web/templates/partials/settings_telegram.html"),
	)
	if err != nil {
		log.Printf("[SETTINGS] Template error: %v", err)
		http.Error(w, "Template Error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":    "System Settings",
		"Settings": settings,
	}

	_ = tmpl.Execute(w, data)
}

// HandleSave processes the form submission to update settings.
func (h *SettingsHandler) HandleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	settings := parseSettingsForm(r)

	// Persist to database
	if err := h.repo.SaveSettings(settings); err != nil {
		log.Printf("[SETTINGS] Failed to save: %v", err)
		http.Error(w, "Failed to save settings", http.StatusInternalServerError)
		return
	}

	log.Println("[SETTINGS] Configuration updated successfully.")
	Redirect(w, r, "/settings", http.StatusSeeOther)
}

// HandleTest sends a test email to the SMTP User themselves to validate the connection.
func (h *SettingsHandler) HandleTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	settings, err := h.repo.GetSettings()
	if err != nil {
		log.Printf("[SETTINGS] Failed to load settings for test: %v", err)
		http.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}

	if settings.SMTPUser == "" {
		http.Error(w, "No account configured. Please Connect Account first.", http.StatusBadRequest)
		return
	}

	targetEmail := settings.SMTPUser
	log.Printf("[SETTINGS] Triggering self-test email to %s via %s:%d...",
		targetEmail, settings.SMTPHost, settings.SMTPPort)

	if h.tester != nil {
		if err := h.tester.TestConnection(settings, targetEmail); err != nil {
			log.Printf("[SETTINGS] Test failed: %v", err)
			http.Error(w, "Test Failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	_, _ = w.Write([]byte("Test email sent to your inbox!"))
}

// HandleTestTelegram sends a test Telegram message to validate the bot token and chat ID.
func (h *SettingsHandler) HandleTestTelegram(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	settings, err := h.repo.GetSettings()
	if err != nil {
		log.Printf("[SETTINGS] Failed to load settings for Telegram test: %v", err)
		http.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}

	if settings.TelegramBotToken == "" {
		http.Error(w, "No Telegram bot configured. Save a bot token first.", http.StatusBadRequest)
		return
	}

	chatID := strings.TrimSpace(r.FormValue("chat_id"))
	if chatID == "" {
		http.Error(w, "Please provide a chat_id to send the test message.", http.StatusBadRequest)
		return
	}

	log.Printf("[SETTINGS] Sending Telegram test to chat_id %s...", chatID)

	if h.tester != nil {
		if err := h.tester.TestTelegramConnection(settings.TelegramBotToken, chatID); err != nil {
			log.Printf("[SETTINGS] Telegram test failed: %v", err)
			http.Error(w, "Test Failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	_, _ = w.Write([]byte("Test message sent! Check your Telegram."))
}
