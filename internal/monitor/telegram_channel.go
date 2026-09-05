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
// File: internal/monitor/telegram_channel.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package monitor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"noxfort-monitor-server/internal/domain"
)

// TelegramChannel handles Telegram MarkdownV2 message formatting and Bot API dispatching.
type TelegramChannel struct {
	client  *http.Client
	baseURL string
}

// NewTelegramChannel creates a new TelegramChannel with default timeout.
func NewTelegramChannel() *TelegramChannel {
	return &TelegramChannel{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// NewTelegramChannelWithClient creates a TelegramChannel with an injected http.Client (for tests).
func NewTelegramChannelWithClient(client *http.Client) *TelegramChannel {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &TelegramChannel{client: client}
}

// NewTelegramChannelWithEndpoint creates a TelegramChannel with an injected client and base API URL.
func NewTelegramChannelWithEndpoint(client *http.Client, baseURL string) *TelegramChannel {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &TelegramChannel{client: client, baseURL: baseURL}
}

// Name returns the channel name.
func (c *TelegramChannel) Name() string {
	return "telegram"
}

// Recipient extracts the Telegram Chat ID destination from the contact.
func (c *TelegramChannel) Recipient(contact *domain.Contact) string {
	if contact == nil {
		return ""
	}
	return contact.TelegramChatID
}

// EscapeMarkdownV2 escapes reserved Telegram MarkdownV2 syntax characters.
func EscapeMarkdownV2(s string) string {
	replacer := strings.NewReplacer(
		"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]",
		"(", "\\(", ")", "\\)", "~", "\\~", "`", "\\`",
		">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-",
		"=", "\\=", "|", "\\|", "{", "\\{", "}", "\\}",
		".", "\\.", "!", "\\!",
	)
	return replacer.Replace(s)
}

// BuildMessage constructs a Telegram MarkdownV2 formatted message with emojis and severity badge.
func (c *TelegramChannel) BuildMessage(identifier string, event *domain.IncomingEvent) string {
	levelEmoji := map[string]string{
		string(domain.LevelCritical): "🔴",
		string(domain.LevelWarning):  "🟡",
		string(domain.LevelInfo):     "🟢",
	}
	emoji := levelEmoji[string(event.Level)]
	if emoji == "" {
		emoji = "⚪"
	}

	return fmt.Sprintf(
		"%s *\\[%s\\] Noxfort Monitor Alert*\n\n"+
			"*Sistema:* `%s`\n"+
			"*Categoria:* %s\n"+
			"*Mensagem:* %s\n"+
			"*Data/Hora:* %s",
		emoji,
		EscapeMarkdownV2(string(event.Level)),
		EscapeMarkdownV2(identifier),
		EscapeMarkdownV2(string(event.Category)),
		EscapeMarkdownV2(event.Message),
		EscapeMarkdownV2(event.OccurredAt.Format("02/01/2006 15:04:05")),
	)
}

// Send dispatches an alert message to the contact's Telegram chat ID.
func (c *TelegramChannel) Send(settings *domain.Settings, contact *domain.Contact, identifier string, event *domain.IncomingEvent) error {
	if settings == nil || settings.TelegramBotToken == "" || contact == nil || contact.TelegramChatID == "" {
		return nil
	}

	msg := c.BuildMessage(identifier, event)
	return c.transmit(settings.TelegramBotToken, contact.TelegramChatID, msg)
}

// Test sends a verification message to validate the bot token and target chat ID.
func (c *TelegramChannel) Test(botToken, chatID string) error {
	msg := "✅ *Noxfort Monitor*\n\nTelegram bot configured successfully\\! Test message received\\."
	return c.transmit(botToken, chatID, msg)
}

func (c *TelegramChannel) transmit(botToken, chatID, text string) error {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://api.telegram.org"
	}
	url := fmt.Sprintf("%s/bot%s/sendMessage", baseURL, botToken)

	payload := map[string]string{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "MarkdownV2",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal telegram payload: %w", err)
	}

	resp, err := c.client.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("telegram API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var result map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&result)
		return fmt.Errorf("telegram API returned %d: %v", resp.StatusCode, result)
	}

	return nil
}
