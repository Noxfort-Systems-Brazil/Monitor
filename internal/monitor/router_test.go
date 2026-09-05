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
// File: internal/monitor/router_test.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package monitor

import (
	"testing"
	"time"

	"noxfort-monitor-server/internal/domain"
)

func TestShouldNotify(t *testing.T) {
	cases := []struct {
		role     string
		category domain.EventCategory
		expected bool
	}{
		{"system_admin", domain.CategoryHardware, true},
		{"admin", domain.CategorySoftware, true},
		{"System Admin", domain.CategoryHardware, true},
		{"technician", domain.CategoryHardware, true},
		{"technician", domain.CategorySoftware, false},
		{"programmer", domain.CategorySoftware, true},
		{"programmer", domain.CategoryHardware, false},
		{"operator", domain.CategoryHardware, false},
		{"unknown", domain.CategorySoftware, false},
	}

	for _, c := range cases {
		got := ShouldNotify(c.role, c.category)
		if got != c.expected {
			t.Errorf("ShouldNotify(%q, %q) = %v, expected %v", c.role, c.category, got, c.expected)
		}
	}
}

func TestIsEligible(t *testing.T) {
	now := time.Now()

	adminContact := &domain.Contact{
		Name:           "Admin",
		Role:           "admin",
		Enabled:        true,
		NotifyCritical: false,
	}

	techContact := &domain.Contact{
		Name:           "Tech",
		Role:           "technician",
		Enabled:        true,
		NotifyCritical: false,
	}

	criticalOnlyContact := &domain.Contact{
		Name:           "OnCall",
		Role:           "admin",
		Enabled:        true,
		NotifyCritical: true,
	}

	disabledContact := &domain.Contact{
		Name:    "Disabled",
		Role:    "admin",
		Enabled: false,
	}

	// 1. Nil checks
	if IsEligible(nil, &domain.IncomingEvent{}) {
		t.Errorf("Nil contact should not be eligible")
	}
	if IsEligible(adminContact, nil) {
		t.Errorf("Nil event should not be eligible")
	}

	// 2. Disabled contact
	if IsEligible(disabledContact, &domain.IncomingEvent{Level: domain.LevelCritical, Category: domain.CategoryHardware}) {
		t.Errorf("Disabled contact must not be eligible")
	}

	// 3. LevelInfo must always be ignored
	infoEvent := &domain.IncomingEvent{Level: domain.LevelInfo, Category: domain.CategoryHardware, OccurredAt: now}
	if IsEligible(adminContact, infoEvent) {
		t.Errorf("INFO events should never be eligible for alerts")
	}

	// 4. Critical-only filter
	warnEvent := &domain.IncomingEvent{Level: domain.LevelWarning, Category: domain.CategoryHardware, OccurredAt: now}
	critEvent := &domain.IncomingEvent{Level: domain.LevelCritical, Category: domain.CategoryHardware, OccurredAt: now}

	if IsEligible(criticalOnlyContact, warnEvent) {
		t.Errorf("Critical-only contact should not receive WARNING event")
	}
	if !IsEligible(criticalOnlyContact, critEvent) {
		t.Errorf("Critical-only contact should receive CRITICAL event")
	}

	// 5. Role vs Category matching
	softWarn := &domain.IncomingEvent{Level: domain.LevelWarning, Category: domain.CategorySoftware, OccurredAt: now}
	hardWarn := &domain.IncomingEvent{Level: domain.LevelWarning, Category: domain.CategoryHardware, OccurredAt: now}

	if IsEligible(techContact, softWarn) {
		t.Errorf("Technician should not be eligible for Software alert")
	}
	if !IsEligible(techContact, hardWarn) {
		t.Errorf("Technician should be eligible for Hardware alert")
	}
}

type mockCustomPolicy struct{}

func (p *mockCustomPolicy) ShouldNotify(role string, category domain.EventCategory) bool {
	// Custom rule: "devops" receives both software and hardware
	if role == "devops" {
		return true
	}
	return false
}

func TestNotificationPolicy_CustomExtension(t *testing.T) {
	// Test standard policy
	if ShouldNotify("devops", domain.CategorySoftware) {
		t.Errorf("Standard policy should not notify 'devops'")
	}

	// Extend via policy injection (OCP)
	origPolicy := defaultPolicy
	defer SetDefaultNotificationPolicy(origPolicy)

	SetDefaultNotificationPolicy(&mockCustomPolicy{})

	if !ShouldNotify("devops", domain.CategorySoftware) {
		t.Errorf("Custom policy should notify 'devops' for software")
	}
	if !ShouldNotify("devops", domain.CategoryHardware) {
		t.Errorf("Custom policy should notify 'devops' for hardware")
	}
	if ShouldNotify("admin", domain.CategoryHardware) {
		t.Errorf("Custom policy does not specify 'admin', should return false")
	}
}
