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
// File: internal/monitor/alerts_test.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package monitor

import (
	"errors"
	"net/smtp"
	"sync"
	"testing"
	"time"

	"noxfort-monitor-server/internal/domain"
)

type mockContactRepoForAlerts struct {
	contacts []domain.Contact
	err      error
}

func (m *mockContactRepoForAlerts) GetAllContacts() ([]domain.Contact, error) {
	return m.contacts, m.err
}
func (m *mockContactRepoForAlerts) CreateContact(c *domain.Contact) error { return nil }
func (m *mockContactRepoForAlerts) UpdateContact(c *domain.Contact) error { return nil }
func (m *mockContactRepoForAlerts) DeleteContact(id int64) error         { return nil }

type mockSettingsRepoForAlerts struct {
	settings *domain.Settings
	err      error
}

func (m *mockSettingsRepoForAlerts) GetSettings() (*domain.Settings, error) {
	return m.settings, m.err
}
func (m *mockSettingsRepoForAlerts) SaveSettings(s *domain.Settings) error { return nil }
func (m *mockSettingsRepoForAlerts) GetSMTPSettings() (*domain.SMTPSettings, error) {
	return nil, nil
}
func (m *mockSettingsRepoForAlerts) SaveSMTPSettings(s *domain.SMTPSettings) error {
	return nil
}

type mockNotificationChannel struct {
	name      string
	mu        sync.Mutex
	sentCalls int
}

func (m *mockNotificationChannel) Name() string {
	return m.name
}

func (m *mockNotificationChannel) Recipient(contact *domain.Contact) string {
	if contact == nil {
		return ""
	}
	return contact.Email
}

func (m *mockNotificationChannel) Send(settings *domain.Settings, contact *domain.Contact, id string, event *domain.IncomingEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentCalls++
	return nil
}

func (m *mockNotificationChannel) getSentCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sentCalls
}

func TestAlertService_TriggerAlert_EndToEnd(t *testing.T) {
	cRepo := &mockContactRepoForAlerts{
		contacts: []domain.Contact{
			{ID: 1, Name: "Admin User", Role: "admin", Enabled: true},
			{ID: 2, Name: "Disabled User", Role: "admin", Enabled: false},
			{ID: 3, Name: "Hardware Tech", Role: "technician", Enabled: true},
		},
	}

	sRepo := &mockSettingsRepoForAlerts{
		settings: &domain.Settings{
			Enabled:          true,
			TelegramBotToken: "token123",
		},
	}

	mockChan1 := &mockNotificationChannel{name: "mock1"}
	mockChan2 := &mockNotificationChannel{name: "mock2"}

	service := NewAlertServiceWithChannels(cRepo, sRepo, mockChan1, mockChan2)

	// 1. Hardware Critical alert: Admin and Hardware Tech are eligible (2 contacts x 2 channels = 4 calls)
	critEvent := &domain.IncomingEvent{
		Level:      domain.LevelCritical,
		Category:   domain.CategoryHardware,
		Message:    "Main Pump RPM dropped below threshold",
		OccurredAt: time.Now(),
	}

	service.TriggerAlert("PUMP-01", critEvent)

	time.Sleep(50 * time.Millisecond)

	if mockChan1.getSentCalls() != 2 {
		t.Errorf("Expected 2 dispatches on mockChan1, got %d", mockChan1.getSentCalls())
	}
	if mockChan2.getSentCalls() != 2 {
		t.Errorf("Expected 2 dispatches on mockChan2, got %d", mockChan2.getSentCalls())
	}

	// 2. Info alert: must be completely ignored (0 additional calls)
	callsBefore := mockChan1.getSentCalls()
	infoEvent := &domain.IncomingEvent{
		Level:      domain.LevelInfo,
		Category:   domain.CategoryHardware,
		Message:    "Routine heartbeat",
		OccurredAt: time.Now(),
	}

	service.TriggerAlert("PUMP-01", infoEvent)
	time.Sleep(30 * time.Millisecond)

	if mockChan1.getSentCalls() != callsBefore {
		t.Errorf("INFO alert should not trigger any dispatches")
	}
}

func TestChannelTester_TestConnection(t *testing.T) {
	// 1. Without email channel (nil)
	testerNilChan := NewChannelTester(nil, nil)
	if err := testerNilChan.TestConnection(&domain.Settings{}, "test@noxfort.com"); err != nil {
		t.Errorf("Expected nil when email channel is nil, got: %v", err)
	}

	// 2. With mockable email channel
	var sentTo string
	emailChan := NewEmailChannelWithSender(func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		if len(to) > 0 {
			sentTo = to[0]
		}
		if sentTo == "fail@noxfort.com" {
			return errors.New("smtp down")
		}
		return nil
	})

	tester := NewChannelTester(emailChan, nil)

	// Success test
	settings := &domain.Settings{SMTPHost: "smtp.local", SMTPPort: 587, SMTPFrom: "alert@local"}
	if err := tester.TestConnection(settings, "admin@noxfort.com"); err != nil {
		t.Errorf("Expected success, got: %v", err)
	}
	if sentTo != "admin@noxfort.com" {
		t.Errorf("Expected recipient 'admin@noxfort.com', got: %s", sentTo)
	}

	// Failure test
	if err := tester.TestConnection(settings, "fail@noxfort.com"); err == nil {
		t.Errorf("Expected error on failing recipient, got nil")
	}
}

func TestChannelTester_TestTelegramConnection(t *testing.T) {
	// 1. Without telegram channel (nil)
	testerNilChan := NewChannelTester(nil, nil)
	if err := testerNilChan.TestTelegramConnection("tok", "chat123"); err != nil {
		t.Errorf("Expected nil when telegram channel is nil, got: %v", err)
	}

	// 2. With telegram channel (using dummy mock URL via endpoint)
	tgChan := NewTelegramChannelWithEndpoint(nil, "http://127.0.0.1:0")
	tester := NewChannelTester(nil, tgChan)

	// Connection error expected on closed local port
	err := tester.TestTelegramConnection("tok", "chat123")
	if err == nil {
		t.Errorf("Expected connection error on dummy address, got nil")
	}
}

func TestAlertService_TriggerAlert_RepositoryErrors(t *testing.T) {
	mockChan := &mockNotificationChannel{name: "mock"}
	event := &domain.IncomingEvent{
		Level:      domain.LevelCritical,
		Category:   domain.CategoryHardware,
		Message:    "Critical failure",
		OccurredAt: time.Now(),
	}

	// 1. Error fetching contacts
	serviceErrContacts := NewAlertServiceWithChannels(
		&mockContactRepoForAlerts{err: errors.New("database locked")},
		&mockSettingsRepoForAlerts{settings: &domain.Settings{Enabled: true}},
		mockChan,
	)
	serviceErrContacts.TriggerAlert("ROBOT-01", event)
	time.Sleep(20 * time.Millisecond)
	if mockChan.getSentCalls() != 0 {
		t.Errorf("Expected no dispatches when contacts query fails")
	}

	// 2. Error fetching settings
	serviceErrSettings := NewAlertServiceWithChannels(
		&mockContactRepoForAlerts{contacts: []domain.Contact{{Role: "admin", Enabled: true}}},
		&mockSettingsRepoForAlerts{err: errors.New("settings corrupt")},
		mockChan,
	)
	serviceErrSettings.TriggerAlert("ROBOT-01", event)
	time.Sleep(20 * time.Millisecond)
	if mockChan.getSentCalls() != 0 {
		t.Errorf("Expected no dispatches when settings query fails")
	}

	// 3. NewAlertService default constructor test
	serviceDefault := NewAlertService(&mockContactRepoForAlerts{}, &mockSettingsRepoForAlerts{})
	if serviceDefault == nil {
		t.Fatalf("Expected non-nil default AlertService")
	}
}
