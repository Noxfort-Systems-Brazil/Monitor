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
// File: internal/monitor/email_channel_test.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package monitor

import (
	"net/smtp"
	"strings"
	"testing"
	"time"

	"noxfort-monitor-server/internal/domain"
)

func TestEmailChannel_FormattingAndMockSend(t *testing.T) {
	sentAddr := ""
	sentFrom := ""
	sentTo := []string{}
	sentMsg := ""

	mockSender := func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		sentAddr = addr
		sentFrom = from
		sentTo = to
		sentMsg = string(msg)
		return nil
	}

	channel := NewEmailChannelWithSender(mockSender)

	if channel.Name() != "email" {
		t.Errorf("Expected channel name 'email', got '%s'", channel.Name())
	}

	testContact := &domain.Contact{Email: "test@example.com"}
	if channel.Recipient(testContact) != "test@example.com" {
		t.Errorf("Expected recipient 'test@example.com', got '%s'", channel.Recipient(testContact))
	}
	if channel.Recipient(nil) != "" {
		t.Errorf("Expected empty recipient for nil contact")
	}

	settings := &domain.Settings{
		SMTPHost: "smtp.example.com",
		SMTPPort: 587,
		SMTPUser: "alerts@example.com",
		SMTPPass: "secret",
		SMTPFrom: "alerts@example.com",
		Enabled:  true,
	}

	contact := &domain.Contact{
		Name:  "Lead Engineer",
		Email: "engineer@example.com",
	}

	event := &domain.IncomingEvent{
		Level:      domain.LevelCritical,
		Category:   domain.CategoryHardware,
		Message:    "Main Boiler Temperature Overheating (185C)",
		OccurredAt: time.Date(2026, 9, 4, 14, 30, 0, 0, time.UTC),
	}

	// 1. Send Alert
	err := channel.Send(settings, contact, "BOILER-01", event)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if sentAddr != "smtp.example.com:587" {
		t.Errorf("Expected addr 'smtp.example.com:587', got '%s'", sentAddr)
	}
	if sentFrom != "alerts@example.com" {
		t.Errorf("Expected from 'alerts@example.com', got '%s'", sentFrom)
	}
	if len(sentTo) != 1 || sentTo[0] != "engineer@example.com" {
		t.Errorf("Expected recipient 'engineer@example.com', got %v", sentTo)
	}
	if !strings.Contains(sentMsg, "Main Boiler Temperature Overheating") {
		t.Errorf("Expected message to contain event text, got:\n%s", sentMsg)
	}

	// 2. Disabled settings should not send
	sentMsg = ""
	disabledSettings := &domain.Settings{Enabled: false}
	_ = channel.Send(disabledSettings, contact, "BOILER-01", event)
	if sentMsg != "" {
		t.Errorf("Disabled settings should not transmit email")
	}

	// 3. Test Connection
	err = channel.Test(settings, "test@example.com")
	if err != nil {
		t.Fatalf("Test connection failed: %v", err)
	}
	if len(sentTo) != 1 || sentTo[0] != "test@example.com" {
		t.Errorf("Expected test recipient 'test@example.com', got %v", sentTo)
	}
}
