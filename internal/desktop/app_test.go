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
// File: internal/desktop/app_test.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package desktop

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDesktopSessionInterceptionAndInjection(t *testing.T) {
	// A mock HTTP handler simulating login, protected page, and logout
	var currentToken string
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/login":
			// Sets session cookie upon login
			http.SetCookie(w, &http.Cookie{
				Name:     "noxfort_session",
				Value:    "test-token-12345",
				Path:     "/",
				HttpOnly: true,
			})
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success": true}`))

		case "/protected":
			// Checks if cookie is present in incoming request
			cookie, err := r.Cookie("noxfort_session")
			if err != nil || cookie.Value == "" {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("unauthorized"))
				return
			}
			currentToken = cookie.Value
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))

		case "/api/auth/logout":
			// Clears session cookie
			http.SetCookie(w, &http.Cookie{
				Name:     "noxfort_session",
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				HttpOnly: true,
			})
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success": true}`))
		}
	})

	app := New(mockHandler, nil)
	assetHandler := app.BuildAssetHandler()

	// 1. Initial request to /protected without cookie -> must be Unauthorized
	req1 := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec1 := httptest.NewRecorder()
	assetHandler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusUnauthorized {
		t.Fatalf("Expected status 401 on initial request, got %d", rec1.Code)
	}

	// 2. Perform login request -> desktop handler must capture session token
	reqLogin := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	recLogin := httptest.NewRecorder()
	assetHandler.ServeHTTP(recLogin, reqLogin)
	if recLogin.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on login, got %d", recLogin.Code)
	}
	if app.getDesktopToken() != "test-token-12345" {
		t.Fatalf("Expected desktopToken to be captured as 'test-token-12345', got %q", app.getDesktopToken())
	}

	// 3. Webview navigates to /protected without sending any Cookie header (simulating WebKitGTK custom URI scheme)
	// The AssetHandler must automatically inject the captured session cookie!
	req2 := httptest.NewRequest(http.MethodGet, "/protected", nil)
	// Explicitly verify req2 has no Cookie header initially
	if req2.Header.Get("Cookie") != "" {
		t.Fatalf("req2 must start without Cookie header")
	}
	rec2 := httptest.NewRecorder()
	assetHandler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on subsequent request with injected cookie, got %d", rec2.Code)
	}
	if currentToken != "test-token-12345" {
		t.Fatalf("Expected handler to receive cookie 'test-token-12345', got %q", currentToken)
	}

	// 4. Perform logout request -> desktop session must be cleared
	reqLogout := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	recLogout := httptest.NewRecorder()
	assetHandler.ServeHTTP(recLogout, reqLogout)
	if recLogout.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on logout, got %d", recLogout.Code)
	}
	if app.getDesktopToken() != "" {
		t.Fatalf("Expected desktopToken to be cleared, got %q", app.getDesktopToken())
	}

	// 5. Subsequent request to /protected must now fail with 401
	req3 := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec3 := httptest.NewRecorder()
	assetHandler.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusUnauthorized {
		t.Fatalf("Expected status 401 after logout, got %d", rec3.Code)
	}
}

func TestDesktopFullscreenEndpoints(t *testing.T) {
	app := New(http.NotFoundHandler(), nil)
	assetHandler := app.BuildAssetHandler()

	// 1. Test Toggle Fullscreen endpoint
	reqToggle := httptest.NewRequest(http.MethodPost, "/api/window/toggle-fullscreen", nil)
	recToggle := httptest.NewRecorder()
	assetHandler.ServeHTTP(recToggle, reqToggle)

	if recToggle.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on /api/window/toggle-fullscreen, got %d", recToggle.Code)
	}
	if ct := recToggle.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Expected Content-Type application/json, got %s", ct)
	}

	// 2. Test Exit Fullscreen endpoint
	reqExit := httptest.NewRequest(http.MethodPost, "/api/window/exit-fullscreen", nil)
	recExit := httptest.NewRecorder()
	assetHandler.ServeHTTP(recExit, reqExit)

	if recExit.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on /api/window/exit-fullscreen, got %d", recExit.Code)
	}
	if ct := recExit.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Expected Content-Type application/json, got %s", ct)
	}
}

