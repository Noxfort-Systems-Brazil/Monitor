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
// File: internal/security/session_test.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package security

import (
	"testing"
	"time"
)

func TestMemorySessionStore_Lifecycle(t *testing.T) {
	ttl := 50 * time.Millisecond
	store := NewMemorySessionStore(ttl)

	// 1. Create session
	token := store.CreateSession("john_doe", "ADMIN")
	if token == "" {
		t.Fatalf("Expected valid non-empty session token")
	}

	// 2. Validate session immediately
	username, role, ok := store.ValidateSession(token)
	if !ok || username != "john_doe" || role != "ADMIN" {
		t.Errorf("Expected valid session, got user=%s, role=%s, ok=%v", username, role, ok)
	}

	// 3. Validate non-existent token
	if _, _, ok := store.ValidateSession("non-existent-token"); ok {
		t.Errorf("Expected validation to fail for non-existent token")
	}

	// 4. Validate empty token
	if _, _, ok := store.ValidateSession(""); ok {
		t.Errorf("Expected validation to fail for empty token")
	}

	// 5. Test TTL expiration
	time.Sleep(70 * time.Millisecond)
	if _, _, ok := store.ValidateSession(token); ok {
		t.Errorf("Expected session to be expired after TTL")
	}
}

func TestMemorySessionStore_Revocation(t *testing.T) {
	store := NewMemorySessionStore(1 * time.Hour)

	token := store.CreateSession("jane_op", "OPERATOR")
	_, _, ok := store.ValidateSession(token)
	if !ok {
		t.Fatalf("Expected session to be valid before revocation")
	}

	store.RevokeSession(token)

	if _, _, ok := store.ValidateSession(token); ok {
		t.Errorf("Expected session to be invalid after explicit revocation")
	}
}
