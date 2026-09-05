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
// File: internal/monitor/tester.go
// Author: Gabriel Moraes
// Date: 2026-09-05 (SOLID Refactor: SRP Separation of Diagnostic Testing)

package monitor

import (
	"noxfort-monitor-server/internal/domain"
)

// ChannelTester validates notification channel credentials and connectivity on demand.
// Decouples diagnostic testing from production alert dispatching (Single Responsibility Principle).
type ChannelTester struct {
	emailChan    *EmailChannel
	telegramChan *TelegramChannel
}

// NewChannelTester initializes the tester with email and telegram channel dependencies.
func NewChannelTester(emailChan *EmailChannel, telegramChan *TelegramChannel) *ChannelTester {
	return &ChannelTester{
		emailChan:    emailChan,
		telegramChan: telegramChan,
	}
}

// TestConnection attempts to send a verification email to validate SMTP configuration.
func (t *ChannelTester) TestConnection(settings *domain.Settings, to string) error {
	if t.emailChan == nil {
		return nil
	}
	return t.emailChan.Test(settings, to)
}

// TestTelegramConnection sends a verification message to validate the bot token and target chat ID.
func (t *ChannelTester) TestTelegramConnection(botToken, chatID string) error {
	if t.telegramChan == nil {
		return nil
	}
	return t.telegramChan.Test(botToken, chatID)
}
