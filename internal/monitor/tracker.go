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
// File: internal/monitor/tracker.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package monitor

import (
	"sync"
)

// SystemStatusTracker manages the online/offline presence state in memory to suppress alert spam.
type SystemStatusTracker struct {
	mu            sync.RWMutex
	offlineStatus map[string]bool
}

// NewSystemStatusTracker initializes an empty tracker.
func NewSystemStatusTracker() *SystemStatusTracker {
	return &SystemStatusTracker{
		offlineStatus: make(map[string]bool),
	}
}

// MarkOffline registers a system as offline.
// It returns true only if the system transitioned from online to offline (first detection).
func (t *SystemStatusTracker) MarkOffline(identifier string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.offlineStatus[identifier] {
		return false
	}

	t.offlineStatus[identifier] = true
	return true
}

// MarkOnline registers a system as recovered.
// It returns true only if the system transitioned from offline to online (recovery detection).
func (t *SystemStatusTracker) MarkOnline(identifier string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.offlineStatus[identifier] {
		return false
	}

	delete(t.offlineStatus, identifier)
	return true
}

// IsOffline checks if an identifier is currently considered offline.
func (t *SystemStatusTracker) IsOffline(identifier string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.offlineStatus[identifier]
}

// Reset clears all tracked presence states.
func (t *SystemStatusTracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.offlineStatus = make(map[string]bool)
}
