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

package http

import (
	"html/template"
	"log"
	"net/http"
	"noxfort-monitor-server/internal/appdir"
	"strconv"

	"noxfort-monitor-server/internal/domain"
	"noxfort-monitor-server/internal/monitor"
)

// SettingsHandler manages the configuration page and actions.
type SettingsHandler struct {
	repo         domain.SettingsRepository
	alertService *monitor.AlertService
}

// NewSettingsHandler creates a handler for system settings.
func NewSettingsHandler(r domain.SettingsRepository, a *monitor.AlertService) *SettingsHandler {
	return &SettingsHandler{
		repo:         r,
		alertService: a,
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

	tmpl.Execute(w, data)
}

// HandleSave processes the form submission to update settings.
func (h *SettingsHandler) HandleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse numeric fields
	port, _ := strconv.Atoi(r.FormValue("smtp_port"))

	// In the simplified UI, we might not have a checkbox for 'enabled',
	// so we assume true if configuring, or handle logic as needed.
	// For now, we assume if user saves credentials, they want it enabled.
	enabled := true

	smtpUser := r.FormValue("smtp_user")

	// AUTO-BINDING:
	// Since the UI doesn't have a separate "Admin Email" field, we assume
	// the SMTP User (the account owner) is also the Admin who receives alerts.
	adminEmail := r.FormValue("admin_email")
	if adminEmail == "" {
		adminEmail = smtpUser
	}

	// Construct domain object from form data
	settings := &domain.Settings{
		SMTPHost:         r.FormValue("smtp_host"),
		SMTPPort:         port,
		SMTPUser:         smtpUser,
		SMTPPass:         r.FormValue("smtp_pass"),
		SMTPFrom:         r.FormValue("smtp_from"),
		AdminEmail:       adminEmail,
		MqttAddress:      r.FormValue("mqtt_address"),
		Enabled:          enabled,
		TelegramBotToken: r.FormValue("telegram_bot_token"),
	}

	// Persist to database
	if err := h.repo.SaveSettings(settings); err != nil {
		log.Printf("[SETTINGS] Failed to save: %v", err)
		http.Error(w, "Failed to save settings", http.StatusInternalServerError)
		return
	}

	log.Println("[SETTINGS] Configuration updated successfully.")
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

// HandleTest sends a test email to the SMTP User themselves.
// This validates the connection and proves the system can send emails.
func (h *SettingsHandler) HandleTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Load Settings from Database
	// The JS sends an empty body, so we rely on what's already saved.
	settings, err := h.repo.GetSettings()
	if err != nil {
		log.Printf("[SETTINGS] Failed to load settings for test: %v", err)
		http.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}

	// 2. Validate Configuration
	if settings.SMTPUser == "" {
		http.Error(w, "No account configured. Please Connect Account first.", http.StatusBadRequest)
		return
	}

	// 3. Set Target: Send to Self (Loopback Test)
	targetEmail := settings.SMTPUser

	log.Printf("[SETTINGS] Triggering self-test email to %s via %s:%d...",
		targetEmail, settings.SMTPHost, settings.SMTPPort)

	// 4. Execute Test
	err = h.alertService.TestConnection(settings, targetEmail)
	if err != nil {
		log.Printf("[SETTINGS] Test failed: %v", err)
		http.Error(w, "Test Failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Test email sent to your inbox!"))
}

// HandleTestTelegram sends a test Telegram message to validate the bot token.
// The caller must provide a chat_id in the POST body (used as the test destination).
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

	// chat_id must be provided in the POST body.
	chatID := r.FormValue("chat_id")
	if chatID == "" {
		http.Error(w, "Please provide a chat_id to send the test message.", http.StatusBadRequest)
		return
	}

	log.Printf("[SETTINGS] Sending Telegram test to chat_id %s...", chatID)

	if err := h.alertService.TestTelegramConnection(settings.TelegramBotToken, chatID); err != nil {
		log.Printf("[SETTINGS] Telegram test failed: %v", err)
		http.Error(w, "Test Failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Test message sent! Check your Telegram."))
}
