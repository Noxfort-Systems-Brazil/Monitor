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
// File: internal/transport/http/redirect_test.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRedirect(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/some-path", nil)
	rec := httptest.NewRecorder()

	Redirect(rec, req, "/login", http.StatusSeeOther)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("Expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}

	loc := rec.Header().Get("Location")
	if loc != "/login" {
		t.Fatalf("Expected Location header '/login', got %q", loc)
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Fatalf("Expected Content-Type text/html, got %q", contentType)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `window.location.replace("/login")`) {
		t.Errorf("Expected body to contain window.location.replace(\"/login\"), got: %s", body)
	}

	if !strings.Contains(body, `meta http-equiv="refresh" content="0; url=/login"`) {
		t.Errorf("Expected body to contain meta refresh, got: %s", body)
	}
}
