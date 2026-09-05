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
// File: internal/monitor/router.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package monitor

import (
	"strings"

	"noxfort-monitor-server/internal/domain"
)

// NotificationPolicy evaluates whether a contact role is eligible to receive alerts for an event category.
// Allows extending the alerting policy without modifying existing router logic (Open/Closed Principle).
type NotificationPolicy interface {
	ShouldNotify(role string, category domain.EventCategory) bool
}

// RoleNotificationPolicy implements standard role-to-category alert routing.
type RoleNotificationPolicy struct{}

// ShouldNotify evaluates standard role matches:
// - Admin receives all categories.
// - Technician receives HARDWARE alerts.
// - Programmer receives SOFTWARE alerts.
func (p *RoleNotificationPolicy) ShouldNotify(role string, category domain.EventCategory) bool {
	role = strings.ToLower(strings.TrimSpace(role))

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

var defaultPolicy NotificationPolicy = &RoleNotificationPolicy{}

// SetDefaultNotificationPolicy allows runtime extension or customization of the notification policy.
func SetDefaultNotificationPolicy(policy NotificationPolicy) {
	if policy != nil {
		defaultPolicy = policy
	}
}

// ShouldNotify determines if a specific role should receive alerts for a specific category using the active policy.
func ShouldNotify(role string, category domain.EventCategory) bool {
	return defaultPolicy.ShouldNotify(role, category)
}

// IsEligible checks all criteria to determine whether a contact should receive a notification for an event:
// - Contact must be enabled.
// - Events with INFO severity are ignored by global policy.
// - If contact requested critical only, event must be CRITICAL.
// - Contact's role must match the event category.
func IsEligible(contact *domain.Contact, event *domain.IncomingEvent) bool {
	if contact == nil || event == nil {
		return false
	}

	if !contact.Enabled {
		return false
	}

	// Global Rule: Never send alerts for INFO level
	if event.Level == domain.LevelInfo {
		return false
	}

	// Severity preference: NotifyCritical requires LevelCritical
	if contact.NotifyCritical && event.Level != domain.LevelCritical {
		return false
	}

	// Role vs Category compatibility
	return ShouldNotify(contact.Role, event.Category)
}
