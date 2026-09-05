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
// File: internal/transport/http/settings_handler_test.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"noxfort-monitor-server/internal/domain"
)

type mockSettingsRepoForHandler struct {
	mu       sync.Mutex
	settings *domain.Settings
}

func (m *mockSettingsRepoForHandler) GetSettings() (*domain.Settings, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.settings == nil {
		return &domain.Settings{}, nil
	}
	return m.settings, nil
}

func (m *mockSettingsRepoForHandler) SaveSettings(s *domain.Settings) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.settings = s
	return nil
}

func (m *mockSettingsRepoForHandler) GetSMTPSettings() (*domain.SMTPSettings, error) {
	return nil, nil
}

func (m *mockSettingsRepoForHandler) SaveSMTPSettings(s *domain.SMTPSettings) error {
	return nil
}

type mockConnectionTester struct {
	mu             sync.Mutex
	emailCalls     int
	telegramCalls  int
	emailTarget    string
	telegramChatID string
	emailErr       error
	telegramErr    error
}

func (m *mockConnectionTester) TestConnection(settings *domain.Settings, to string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.emailCalls++
	m.emailTarget = to
	return m.emailErr
}

func (m *mockConnectionTester) TestTelegramConnection(botToken, chatID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.telegramCalls++
	m.telegramChatID = chatID
	return m.telegramErr
}

func TestSettingsHandler_HandleSave(t *testing.T) {
	repo := &mockSettingsRepoForHandler{}
	tester := &mockConnectionTester{}
	handler := NewSettingsHandler(repo, tester)

	form := url.Values{}
	form.Set("smtp_host", "smtp.office365.com")
	form.Set("smtp_port", "587")
	form.Set("smtp_user", "plant_admin@factory.com")
	form.Set("smtp_pass", "secretpass")
	form.Set("smtp_from", "alerts@factory.com")
	form.Set("admin_email", "manager@factory.com")
	form.Set("mqtt_address", "tcp://192.168.1.50:1883")
	form.Set("telegram_bot_token", "123456:ABC-DEF")

	req := httptest.NewRequest(http.MethodPost, "/settings/save", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	handler.HandleSave(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("Expected redirect status 303, got %d", rr.Code)
	}

	saved := repo.settings
	if saved == nil {
		t.Fatalf("Settings were not saved to repository")
	}

	if saved.SMTPHost != "smtp.office365.com" || saved.SMTPPort != 587 {
		t.Errorf("SMTP Host/Port mismatch: host=%s port=%d", saved.SMTPHost, saved.SMTPPort)
	}
	if saved.SMTPUser != "plant_admin@factory.com" || saved.AdminEmail != "manager@factory.com" {
		t.Errorf("User/AdminEmail mismatch: user=%s admin=%s", saved.SMTPUser, saved.AdminEmail)
	}
	if saved.MqttAddress != "tcp://192.168.1.50:1883" || saved.TelegramBotToken != "123456:ABC-DEF" {
		t.Errorf("MQTT/Telegram mismatch: mqtt=%s tg=%s", saved.MqttAddress, saved.TelegramBotToken)
	}
}

func TestSettingsHandler_HandleTest_Email(t *testing.T) {
	// 1. Unconfigured account -> 400
	repoEmpty := &mockSettingsRepoForHandler{settings: &domain.Settings{SMTPUser: ""}}
	tester := &mockConnectionTester{}
	handlerEmpty := NewSettingsHandler(repoEmpty, tester)

	reqEmpty := httptest.NewRequest(http.MethodPost, "/settings/test", nil)
	rrEmpty := httptest.NewRecorder()
	handlerEmpty.HandleTest(rrEmpty, reqEmpty)

	if rrEmpty.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 when SMTP is unconfigured, got %d", rrEmpty.Code)
	}

	// 2. Successful test -> 200
	repoValid := &mockSettingsRepoForHandler{
		settings: &domain.Settings{
			SMTPHost: "smtp.mail.com",
			SMTPPort: 587,
			SMTPUser: "engineer@mail.com",
		},
	}
	handlerValid := NewSettingsHandler(repoValid, tester)

	reqValid := httptest.NewRequest(http.MethodPost, "/settings/test", nil)
	rrValid := httptest.NewRecorder()
	handlerValid.HandleTest(rrValid, reqValid)

	if rrValid.Code != http.StatusOK {
		t.Errorf("Expected 200 on valid email test, got %d", rrValid.Code)
	}
	if tester.emailCalls != 1 || tester.emailTarget != "engineer@mail.com" {
		t.Errorf("Expected test call to engineer@mail.com, got calls=%d target=%s", tester.emailCalls, tester.emailTarget)
	}

	// 3. Failed connection -> 500
	tester.emailErr = errors.New("auth failed")
	reqFail := httptest.NewRequest(http.MethodPost, "/settings/test", nil)
	rrFail := httptest.NewRecorder()
	handlerValid.HandleTest(rrFail, reqFail)

	if rrFail.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 when email test fails, got %d", rrFail.Code)
	}
}

func TestSettingsHandler_HandleTest_Telegram(t *testing.T) {
	// 1. Unconfigured bot token -> 400
	repoNoToken := &mockSettingsRepoForHandler{settings: &domain.Settings{TelegramBotToken: ""}}
	tester := &mockConnectionTester{}
	handlerNoToken := NewSettingsHandler(repoNoToken, tester)

	reqNoToken := httptest.NewRequest(http.MethodPost, "/settings/test-telegram", nil)
	rrNoToken := httptest.NewRecorder()
	handlerNoToken.HandleTestTelegram(rrNoToken, reqNoToken)

	if rrNoToken.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 when bot token is empty, got %d", rrNoToken.Code)
	}

	// 2. Configured bot token but missing chat_id -> 400
	repoValid := &mockSettingsRepoForHandler{
		settings: &domain.Settings{TelegramBotToken: "valid-bot-token"},
	}
	handlerValid := NewSettingsHandler(repoValid, tester)

	reqNoChat := httptest.NewRequest(http.MethodPost, "/settings/test-telegram", nil)
	rrNoChat := httptest.NewRecorder()
	handlerValid.HandleTestTelegram(rrNoChat, reqNoChat)

	if rrNoChat.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 when chat_id is missing, got %d", rrNoChat.Code)
	}

	// 3. Valid bot token and chat_id -> 200
	form := url.Values{}
	form.Set("chat_id", "987654321")
	reqSuccess := httptest.NewRequest(http.MethodPost, "/settings/test-telegram", strings.NewReader(form.Encode()))
	reqSuccess.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rrSuccess := httptest.NewRecorder()

	handlerValid.HandleTestTelegram(rrSuccess, reqSuccess)

	if rrSuccess.Code != http.StatusOK {
		t.Errorf("Expected 200 on successful telegram test, got %d", rrSuccess.Code)
	}
	if tester.telegramCalls != 1 || tester.telegramChatID != "987654321" {
		t.Errorf("Expected telegram test to chat_id 987654321, got calls=%d chat=%s", tester.telegramCalls, tester.telegramChatID)
	}

	// 4. Failed telegram transmission -> 500
	tester.telegramErr = errors.New("chat not found")
	reqFail := httptest.NewRequest(http.MethodPost, "/settings/test-telegram", strings.NewReader(form.Encode()))
	reqFail.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rrFail := httptest.NewRecorder()

	handlerValid.HandleTestTelegram(rrFail, reqFail)

	if rrFail.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 on telegram test error, got %d", rrFail.Code)
	}
}
