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
// File: internal/monitor/telegram_channel_test.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package monitor

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"noxfort-monitor-server/internal/domain"
)

func TestEscapeMarkdownV2(t *testing.T) {
	input := "Alert: Server_01 is down! (Error: code-404. Check [docs]*)"
	escaped := EscapeMarkdownV2(input)

	forbidden := []string{"_", "*", "[", "]", "(", ")", ".", "!", "-"}
	for _, char := range forbidden {
		if strings.Contains(escaped, "\\"+char) == false && strings.Contains(input, char) {
			t.Errorf("Character %q was not properly escaped in %s", char, escaped)
		}
	}
}

func TestTelegramChannel_Name(t *testing.T) {
	channel := NewTelegramChannel()
	if channel.Name() != "telegram" {
		t.Errorf("Expected channel name 'telegram', got '%s'", channel.Name())
	}
	channelWithClient := NewTelegramChannelWithClient(http.DefaultClient)
	if channelWithClient.Name() != "telegram" {
		t.Errorf("Expected channel name 'telegram', got '%s'", channelWithClient.Name())
	}

	testContact := &domain.Contact{TelegramChatID: "123456789"}
	if channel.Recipient(testContact) != "123456789" {
		t.Errorf("Expected recipient '123456789', got '%s'", channel.Recipient(testContact))
	}
	if channel.Recipient(nil) != "" {
		t.Errorf("Expected empty recipient for nil contact")
	}
}

func TestTelegramChannel_SendAndTest_WithMockServer(t *testing.T) {
	var receivedBody map[string]string
	serverStatus := http.StatusOK

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/bottoken123/sendMessage") {
			t.Errorf("Unexpected URL path: %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(serverStatus)
		if serverStatus == http.StatusOK {
			_, _ = w.Write([]byte(`{"ok": true, "result": {"message_id": 101}}`))
		} else {
			_, _ = w.Write([]byte(`{"ok": false, "description": "Bad Request: chat not found"}`))
		}
	}))
	defer ts.Close()

	channel := NewTelegramChannelWithEndpoint(ts.Client(), ts.URL)

	// 1. Valid Alert Send
	settings := &domain.Settings{
		TelegramBotToken: "token123",
	}
	contact := &domain.Contact{
		TelegramChatID: "998877",
	}
	event := &domain.IncomingEvent{
		Level:      domain.LevelCritical,
		Category:   domain.CategoryHardware,
		Message:    "Voltage drop detected on Phase 2",
		OccurredAt: time.Date(2026, 9, 4, 15, 0, 0, 0, time.UTC),
	}

	err := channel.Send(settings, contact, "TRANSFORMER-03", event)
	if err != nil {
		t.Fatalf("Expected successful Send, got: %v", err)
	}

	if receivedBody["chat_id"] != "998877" {
		t.Errorf("Expected chat_id '998877', got '%s'", receivedBody["chat_id"])
	}
	if receivedBody["parse_mode"] != "MarkdownV2" {
		t.Errorf("Expected parse_mode 'MarkdownV2', got '%s'", receivedBody["parse_mode"])
	}
	if !strings.Contains(receivedBody["text"], "🔴") || !strings.Contains(receivedBody["text"], "Voltage drop detected") {
		t.Errorf("Expected message to contain critical indicator and text, got: %s", receivedBody["text"])
	}

	// 2. Guard conditions (nil settings, empty bot token, nil contact, empty chat id)
	receivedBody = nil
	_ = channel.Send(nil, contact, "DEV-01", event)
	_ = channel.Send(&domain.Settings{TelegramBotToken: ""}, contact, "DEV-01", event)
	_ = channel.Send(settings, nil, "DEV-01", event)
	_ = channel.Send(settings, &domain.Contact{TelegramChatID: ""}, "DEV-01", event)
	if receivedBody != nil {
		t.Errorf("Expected no HTTP request when credentials are missing")
	}

	// 3. Test verification message
	err = channel.Test("token123", "998877")
	if err != nil {
		t.Fatalf("Expected successful Test, got: %v", err)
	}
	if !strings.Contains(receivedBody["text"], "Telegram bot configured successfully") {
		t.Errorf("Expected test message content, got: %s", receivedBody["text"])
	}

	// 4. Server error handling
	serverStatus = http.StatusBadRequest
	err = channel.Test("token123", "998877")
	if err == nil {
		t.Fatalf("Expected error when telegram returns 400")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("Expected error to mention status 400, got: %v", err)
	}
}
