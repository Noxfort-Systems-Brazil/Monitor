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
// File: internal/monitor/email_channel.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package monitor

import (
	"fmt"
	"net/smtp"

	"noxfort-monitor-server/internal/domain"
)

// SMTPSenderFunc represents the low-level SMTP send function, allowing test mocking.
type SMTPSenderFunc func(addr string, a smtp.Auth, from string, to []string, msg []byte) error

// EmailChannel handles email formatting and SMTP delivery.
type EmailChannel struct {
	sendMail SMTPSenderFunc
}

// NewEmailChannel creates a new EmailChannel using standard net/smtp.SendMail.
func NewEmailChannel() *EmailChannel {
	return &EmailChannel{sendMail: smtp.SendMail}
}

// NewEmailChannelWithSender creates an EmailChannel with a custom SMTP sender (for testing).
func NewEmailChannelWithSender(sender SMTPSenderFunc) *EmailChannel {
	return &EmailChannel{sendMail: sender}
}

// Name returns the channel name.
func (c *EmailChannel) Name() string {
	return "email"
}

// Recipient extracts the email destination address from the contact.
func (c *EmailChannel) Recipient(contact *domain.Contact) string {
	if contact == nil {
		return ""
	}
	return contact.Email
}

// BuildSubject constructs a standardized alert email subject.
func (c *EmailChannel) BuildSubject(identifier string, event *domain.IncomingEvent) string {
	return fmt.Sprintf("[%s] %s - %s: %s", event.Level, event.Category, identifier, event.Message)
}

// BuildBody formats the plain-text body of the incident email.
func (c *EmailChannel) BuildBody(identifier string, event *domain.IncomingEvent) string {
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

// Send formats and sends an alert email to the contact if SMTP and recipient email are configured.
func (c *EmailChannel) Send(settings *domain.Settings, contact *domain.Contact, identifier string, event *domain.IncomingEvent) error {
	if settings == nil || !settings.Enabled || contact == nil || contact.Email == "" {
		return nil
	}

	subject := c.BuildSubject(identifier, event)
	body := c.BuildBody(identifier, event)

	return c.transmit(settings, contact.Email, subject, body)
}

// Test sends a verification email to check SMTP connectivity.
func (c *EmailChannel) Test(settings *domain.Settings, to string) error {
	subject := "Noxfort Monitor: Test Connection"
	body := "This is a test email to verify your SMTP configuration."
	return c.transmit(settings, to, subject, body)
}

func (c *EmailChannel) transmit(settings *domain.Settings, to, subject, body string) error {
	auth := smtp.PlainAuth("", settings.SMTPUser, settings.SMTPPass, settings.SMTPHost)

	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"From: %s\r\n"+
		"Subject: %s\r\n"+
		"\r\n"+
		"%s\r\n", to, settings.SMTPFrom, subject, body))

	addr := fmt.Sprintf("%s:%d", settings.SMTPHost, settings.SMTPPort)
	return c.sendMail(addr, auth, settings.SMTPFrom, []string{to}, msg)
}
