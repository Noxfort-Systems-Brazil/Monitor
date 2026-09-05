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
// File: internal/transport/http/browser_handler_test.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestBrowserHandler_HandleOpenExternal(t *testing.T) {
	openedURL := ""
	mockOpener := func(targetURL string) error {
		openedURL = targetURL
		return nil
	}

	handler := NewBrowserHandlerWithOpener(mockOpener)

	// 1. Invalid or dangerous URLs (must be rejected with 400)
	invalidURLs := []string{
		"",
		"javascript:alert(1)",
		"file:///etc/passwd",
		"ftp://server.com",
		"data:text/html,test",
	}

	for _, badURL := range invalidURLs {
		req := httptest.NewRequest(http.MethodGet, "/api/open-external?url="+url.QueryEscape(badURL), nil)
		rr := httptest.NewRecorder()
		handler.HandleOpenExternal(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for unsafe URL %q, got %d", badURL, rr.Code)
		}
	}

	// 2. Valid HTTP URL
	validURL := "https://noxfort.com/docs"
	reqValid := httptest.NewRequest(http.MethodGet, "/api/open-external?url="+url.QueryEscape(validURL), nil)
	rrValid := httptest.NewRecorder()
	handler.HandleOpenExternal(rrValid, reqValid)

	if rrValid.Code != http.StatusOK {
		t.Errorf("Expected 200 on valid HTTPS URL, got %d", rrValid.Code)
	}
	if openedURL != validURL {
		t.Errorf("Expected opener to receive %q, got %q", validURL, openedURL)
	}

	// 3. Opener failure
	failingHandler := NewBrowserHandlerWithOpener(func(targetURL string) error {
		return errors.New("command failed")
	})
	reqFail := httptest.NewRequest(http.MethodGet, "/api/open-external?url="+url.QueryEscape(validURL), nil)
	rrFail := httptest.NewRecorder()
	failingHandler.HandleOpenExternal(rrFail, reqFail)

	if rrFail.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 when opener fails, got %d", rrFail.Code)
	}
}
