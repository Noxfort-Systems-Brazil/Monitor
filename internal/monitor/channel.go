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
// File: internal/monitor/channel.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package monitor

import (
	"noxfort-monitor-server/internal/domain"
)

// NotificationChannel represents an alert delivery medium (e.g. Email, Telegram, Slack).
type NotificationChannel interface {
	Name() string
	Recipient(contact *domain.Contact) string
	Send(settings *domain.Settings, contact *domain.Contact, identifier string, event *domain.IncomingEvent) error
}

// AlertDispatcher defines the capability to dispatch alert events to subscribers.
type AlertDispatcher interface {
	TriggerAlert(identifier string, event *domain.IncomingEvent)
}
