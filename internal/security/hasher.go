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
// File: internal/security/hasher.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package security

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

const (
	PBKDF2Iterations = 100000
	SaltLength       = 16
	KeyLength        = 32
)

// PasswordHasher abstracts password hashing and verification algorithms.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, storedHash string) bool
}

// PBKDF2Hasher implements PasswordHasher using PBKDF2 HMAC-SHA256.
type PBKDF2Hasher struct {
	iterations int
	saltLength int
	keyLength  int
}

// NewPBKDF2Hasher creates a new PBKDF2Hasher instance with default security parameters.
func NewPBKDF2Hasher() *PBKDF2Hasher {
	return &PBKDF2Hasher{
		iterations: PBKDF2Iterations,
		saltLength: SaltLength,
		keyLength:  KeyLength,
	}
}

// Hash creates a PBKDF2 HMAC-SHA256 password hash in hex(salt):hex(hash) format.
func (h *PBKDF2Hasher) Hash(password string) (string, error) {
	salt := make([]byte, h.saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate random salt: %w", err)
	}

	hash := pbkdf2.Key([]byte(password), salt, h.iterations, h.keyLength, sha256.New)
	return fmt.Sprintf("%s:%s", hex.EncodeToString(salt), hex.EncodeToString(hash)), nil
}

// Verify verifies a plaintext password against a hex(salt):hex(hash) PBKDF2 string.
func (h *PBKDF2Hasher) Verify(password, storedHashStr string) bool {
	parts := strings.Split(storedHashStr, ":")
	if len(parts) != 2 {
		return false
	}

	salt, err := hex.DecodeString(parts[0])
	if err != nil {
		return false
	}

	expectedHash, err := hex.DecodeString(parts[1])
	if err != nil {
		return false
	}

	computedHash := pbkdf2.Key([]byte(password), salt, h.iterations, h.keyLength, sha256.New)
	return subtle.ConstantTimeCompare(computedHash, expectedHash) == 1
}

var defaultHasher = NewPBKDF2Hasher()

// HashPassword creates a PBKDF2 HMAC-SHA256 password hash using the default hasher.
func HashPassword(password string) (string, error) {
	return defaultHasher.Hash(password)
}

// VerifyPassword verifies a plaintext password against a hex(salt):hex(hash) PBKDF2 string using the default hasher.
func VerifyPassword(password, storedHashStr string) bool {
	return defaultHasher.Verify(password, storedHashStr)
}
