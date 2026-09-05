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
// Modified: 2026-09-04 (SOLID Refactor)

package monitor

import (
	"log"

	"noxfort-monitor-server/internal/domain"
)

// AlertService orchestrates the retrieval of settings/contacts and dispatches alerts via configured channels.
type AlertService struct {
	contactRepo  domain.ContactRepository
	settingsRepo domain.SettingsRepository
	auditRepo    domain.AuditRepository
	channels     []NotificationChannel
}

// SetAuditRepository attaches the audit repository for recording notification delivery logs.
func (s *AlertService) SetAuditRepository(repo domain.AuditRepository) {
	s.auditRepo = repo
}

// NewAlertService creates a new instance of the AlertService with default or injected channels.
func NewAlertService(cRepo domain.ContactRepository, sRepo domain.SettingsRepository, channels ...NotificationChannel) *AlertService {
	if len(channels) == 0 {
		channels = []NotificationChannel{NewEmailChannel(), NewTelegramChannel()}
	}

	return &AlertService{
		contactRepo:  cRepo,
		settingsRepo: sRepo,
		channels:     channels,
	}
}

// NewAlertServiceWithChannels creates an AlertService with custom channels (useful for testing or extensions).
func NewAlertServiceWithChannels(
	cRepo domain.ContactRepository,
	sRepo domain.SettingsRepository,
	channels ...NotificationChannel,
) *AlertService {
	return NewAlertService(cRepo, sRepo, channels...)
}

// TriggerAlert processes an incoming event, evaluates routing rules, and dispatches to eligible contacts.
func (s *AlertService) TriggerAlert(identifier string, event *domain.IncomingEvent) {
	if event == nil {
		return
	}

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

	// 3. Smart Routing & Channel Dispatch
	sentCount := 0
	for _, contact := range contacts {
		target := contact
		if !IsEligible(&target, event) {
			continue
		}

		for _, ch := range s.channels {
			go func(channel NotificationChannel, c domain.Contact) {
				status := "SENT"
				errReason := ""
				if err := channel.Send(settings, &c, identifier, event); err != nil {
					status = "FAILED"
					errReason = err.Error()
					log.Printf("[ALERTS] Failed to send alert via %s to contact '%s': %v",
						channel.Name(), c.Name, err)
				}

				if s.auditRepo != nil {
					recipient := channel.Recipient(&c)
					_ = s.auditRepo.SaveAlertDispatchLog(&domain.AlertDispatchLog{
						Channel:      channel.Name(),
						Recipient:    recipient,
						Role:         c.Role,
						Status:       status,
						ErrorReason:  errReason,
						DispatchedAt: event.OccurredAt,
					})
				}
			}(ch, target)
		}
		sentCount++
	}

	if sentCount > 0 {
		log.Printf("[ALERTS] Dispatching incident '%s' to %d eligible recipients across %d channels.",
			event.Message, sentCount, len(s.channels))
	}
}
