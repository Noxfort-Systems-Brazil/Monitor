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
// File: internal/security/session.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package security

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// SessionTTL is the default lifetime for user sessions.
const SessionTTL = 24 * time.Hour

// Session represents an active authenticated user session.
type Session struct {
	Token     string
	Username  string
	Role      string
	ExpiresAt time.Time
}

// SessionStore abstracts session persistence, retrieval, and revocation.
type SessionStore interface {
	CreateSession(username, role string) string
	ValidateSession(token string) (string, string, bool)
	RevokeSession(token string)
}

// MemorySessionStore manages active user sessions in memory with thread safety.
type MemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]Session
	ttl      time.Duration
}

// NewMemorySessionStore creates a new in-memory session store.
func NewMemorySessionStore(ttl time.Duration) *MemorySessionStore {
	if ttl <= 0 {
		ttl = SessionTTL
	}
	return &MemorySessionStore{
		sessions: make(map[string]Session),
		ttl:      ttl,
	}
}

// CreateSession generates a cryptographically secure token and stores the session.
func (s *MemorySessionStore) CreateSession(username, role string) string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	token := hex.EncodeToString(b)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[token] = Session{
		Token:     token,
		Username:  username,
		Role:      role,
		ExpiresAt: time.Now().Add(s.ttl),
	}

	return token
}

// ValidateSession checks if a session exists and is still valid within TTL.
func (s *MemorySessionStore) ValidateSession(token string) (string, string, bool) {
	if token == "" {
		return "", "", false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[token]
	if !ok {
		return "", "", false
	}

	if time.Now().After(sess.ExpiresAt) {
		delete(s.sessions, token)
		return "", "", false
	}

	return sess.Username, sess.Role, true
}

// RevokeSession deletes an active session from memory.
func (s *MemorySessionStore) RevokeSession(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}
