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
// File: internal/security/hasher_test.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package security

import (
	"testing"
)

func TestPBKDF2Hasher_HashAndVerify(t *testing.T) {
	hasher := NewPBKDF2Hasher()

	password := "TestPassword@2026"

	// 1. Hash password
	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}
	if hash == "" {
		t.Fatalf("Expected non-empty hash string")
	}

	// 2. Verify with correct password
	if !hasher.Verify(password, hash) {
		t.Errorf("Expected password verification to succeed")
	}

	// 3. Verify with wrong password
	if hasher.Verify("WrongPassword!123", hash) {
		t.Errorf("Expected verification to fail with wrong password")
	}

	// 4. Verify with malformed hash
	if hasher.Verify(password, "invalid_format") {
		t.Errorf("Expected verification to fail with malformed hash")
	}
	if hasher.Verify(password, "nothex:nothex") {
		t.Errorf("Expected verification to fail with non-hex hash")
	}
}

func TestPBKDF2Hasher_UniqueSalts(t *testing.T) {
	hasher := NewPBKDF2Hasher()
	password := "SamePasswordAcrossUsers"

	hash1, err1 := hasher.Hash(password)
	hash2, err2 := hasher.Hash(password)

	if err1 != nil || err2 != nil {
		t.Fatalf("Hashing failed: %v, %v", err1, err2)
	}

	if hash1 == hash2 {
		t.Errorf("Expected distinct salts to produce different hash strings for the same password")
	}

	if !hasher.Verify(password, hash1) || !hasher.Verify(password, hash2) {
		t.Errorf("Both distinct hashes must verify correctly with the password")
	}
}
