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
// File: internal/monitor/alerts.go
// Author: Gabriel Moraes
// Date: 2026-01-19

package monitor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"strings"

	"noxfort-monitor-server/internal/domain"
)

// AlertService manages the dispatch of notifications based on roles and categories.
type AlertService struct {
	contactRepo  domain.ContactRepository
	settingsRepo domain.SettingsRepository
}

// NewAlertService creates a new instance of the AlertService.
func NewAlertService(cRepo domain.ContactRepository, sRepo domain.SettingsRepository) *AlertService {
	return &AlertService{
		contactRepo:  cRepo,
		settingsRepo: sRepo,
	}
}

// TriggerAlert processes an incoming event and dispatches notifications if necessary.
func (s *AlertService) TriggerAlert(identifier string, event *domain.IncomingEvent) {
	// 1. Fetch Global Settings
	settings, err := s.settingsRepo.GetSettings()
	if err != nil {
		log.Printf("[ALERTS] Failed to fetch settings: %v", err)
		return
	}

	// 2. Fetch Contacts
	contacts, err := s.contactRepo.GetAllContacts()
	if err != nil {
		log.Printf("[ALERTS] Failed to fetch contacts: %v", err)
		return
	}

	// 3. Compose the messages
	subject := fmt.Sprintf("[%s] %s - %s: %s", event.Level, event.Category, identifier, event.Message)
	emailBody := s.buildEmailBody(identifier, event)
	telegramMsg := s.buildTelegramMessage(identifier, event)

	// 4. Smart Routing Dispatch
	sentCount := 0
	for _, contact := range contacts {
		if !contact.Enabled {
			continue
		}

		// Regra Global: Nunca enviar alertas para nível INFO.
		if event.Level == domain.LevelInfo {
			continue
		}

		// A. Check Severity Preference
		if contact.NotifyCritical && event.Level != domain.LevelCritical {
			continue
		}

		// B. Check Role vs Category Compatibility
		if !s.shouldNotify(contact.Role, event.Category) {
			continue
		}

		// C. Send Email (if SMTP is configured)
		if settings.Enabled && contact.Email != "" {
			go func(email string) {
				if err := s.sendEmail(settings, email, subject, emailBody); err != nil {
					log.Printf("[ALERTS] Failed to send email to %s: %v", email, err)
				}
			}(contact.Email)
		}

		// D. Send Telegram (if bot token + contact chat ID are configured)
		if settings.TelegramBotToken != "" && contact.TelegramChatID != "" {
			go func(chatID string) {
				if err := s.sendTelegram(settings.TelegramBotToken, chatID, telegramMsg); err != nil {
					log.Printf("[ALERTS] Failed to send Telegram to %s: %v", chatID, err)
				}
			}(contact.TelegramChatID)
		}

		sentCount++
	}

	if sentCount > 0 {
		log.Printf("[ALERTS] Dispatching incident '%s' to %d recipients.", event.Message, sentCount)
	}
}

// shouldNotify determines if a specific Role should receive a specific Category of alert.
func (s *AlertService) shouldNotify(role string, category domain.EventCategory) bool {
	role = strings.ToLower(role)

	// 1. System Admin (Receives EVERYTHING)
	if role == "system_admin" || role == "admin" || role == "system admin" {
		return true
	}

	// 2. Technician (Receives only HARDWARE)
	if role == "technician" && category == domain.CategoryHardware {
		return true
	}

	// 3. Programmer (Receives only SOFTWARE)
	if role == "programmer" && category == domain.CategorySoftware {
		return true
	}

	return false
}

// TestConnection attempts to send a test email.
func (s *AlertService) TestConnection(settings *domain.Settings, to string) error {
	subject := "Noxfort Monitor: Test Connection"
	body := "This is a test email to verify your SMTP configuration."
	return s.sendEmail(settings, to, subject, body)
}

// TestTelegramConnection sends a test message to validate the bot token and a given chat ID.
func (s *AlertService) TestTelegramConnection(botToken, chatID string) error {
	msg := "✅ *Noxfort Monitor*\n\nTelegram bot configured successfully\\! Test message received\\."
	return s.sendTelegram(botToken, chatID, msg)
}

// buildEmailBody formats the alert email body.
func (s *AlertService) buildEmailBody(identifier string, event *domain.IncomingEvent) string {
	return fmt.Sprintf(
		"NOXFORT MONITOR - RELATÓRIO DE INCIDENTE\n"+
			"--------------------------------------------------\n"+
			"Categoria:    %s\n"+
			"Sistema:      %s\n"+
			"Gravidade:    %s\n"+
			"Data/Hora:    %s\n"+
			"--------------------------------------------------\n"+
			"\n"+
			"MENSAGEM:\n"+
			"%s\n",
		event.Category,
		identifier,
		event.Level,
		event.OccurredAt.Format("02/01/2006 15:04:05"),
		event.Message,
	)
}

// buildTelegramMessage formats the alert as a Telegram MarkdownV2 message.
func (s *AlertService) buildTelegramMessage(identifier string, event *domain.IncomingEvent) string {
	levelEmoji := map[string]string{
		string(domain.LevelCritical): "🔴",
		string(domain.LevelWarning):  "🟡",
		string(domain.LevelInfo):     "🟢",
	}
	emoji := levelEmoji[string(event.Level)]
	if emoji == "" {
		emoji = "⚪"
	}

	// Escape special chars for MarkdownV2
	esc := func(s string) string {
		replacer := strings.NewReplacer(
			"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]",
			"(", "\\(", ")", "\\)", "~", "\\~", "`", "\\`",
			">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-",
			"=", "\\=", "|", "\\|", "{", "\\{", "}", "\\}",
			".", "\\.", "!", "\\!",
		)
		return replacer.Replace(s)
	}

	return fmt.Sprintf(
		"%s *\\[%s\\] Noxfort Monitor Alert*\n\n"+
			"*Sistema:* `%s`\n"+
			"*Categoria:* %s\n"+
			"*Mensagem:* %s\n"+
			"*Data/Hora:* %s",
		emoji,
		esc(string(event.Level)),
		esc(identifier),
		esc(string(event.Category)),
		esc(event.Message),
		esc(event.OccurredAt.Format("02/01/2006 15:04:05")),
	)
}

// sendTelegram dispatches a message via the Telegram Bot API.
func (s *AlertService) sendTelegram(botToken, chatID, text string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	payload := map[string]string{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "MarkdownV2",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal telegram payload: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("telegram API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		return fmt.Errorf("telegram API returned %d: %v", resp.StatusCode, result)
	}

	return nil
}

// sendEmail performs the actual SMTP transmission.
func (s *AlertService) sendEmail(settings *domain.Settings, to, subject, body string) error {
	auth := smtp.PlainAuth("", settings.SMTPUser, settings.SMTPPass, settings.SMTPHost)

	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"From: %s\r\n"+
		"Subject: %s\r\n"+
		"\r\n"+
		"%s\r\n", to, settings.SMTPFrom, subject, body))

	addr := fmt.Sprintf("%s:%d", settings.SMTPHost, settings.SMTPPort)
	return smtp.SendMail(addr, auth, settings.SMTPFrom, []string{to}, msg)
}
