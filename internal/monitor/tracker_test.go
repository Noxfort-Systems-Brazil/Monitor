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
// File: internal/monitor/tracker_test.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package monitor

import (
	"sync"
	"testing"
)

func TestSystemStatusTracker_StateTransitions(t *testing.T) {
	tracker := NewSystemStatusTracker()

	systemID := "ROBOT-ARM-01"

	// 1. Initial state: not offline
	if tracker.IsOffline(systemID) {
		t.Errorf("Expected %s not to be offline initially", systemID)
	}

	// 2. MarkOffline first detection -> returns true
	if !tracker.MarkOffline(systemID) {
		t.Errorf("Expected first MarkOffline to return true (transition from online to offline)")
	}
	if !tracker.IsOffline(systemID) {
		t.Errorf("Expected %s to be marked as offline", systemID)
	}

	// 3. MarkOffline subsequent call (anti-spam / deduplication) -> returns false
	if tracker.MarkOffline(systemID) {
		t.Errorf("Expected subsequent MarkOffline to return false (already offline)")
	}

	// 4. MarkOnline recovery detection -> returns true
	if !tracker.MarkOnline(systemID) {
		t.Errorf("Expected first MarkOnline to return true (recovery transition)")
	}
	if tracker.IsOffline(systemID) {
		t.Errorf("Expected %s to no longer be marked as offline", systemID)
	}

	// 5. MarkOnline subsequent call -> returns false (already online)
	if tracker.MarkOnline(systemID) {
		t.Errorf("Expected subsequent MarkOnline to return false (already online)")
	}

	// 6. Reset clears all states
	tracker.MarkOffline("SYSTEM-A")
	tracker.MarkOffline("SYSTEM-B")
	if !tracker.IsOffline("SYSTEM-A") || !tracker.IsOffline("SYSTEM-B") {
		t.Fatalf("Expected both systems to be offline before reset")
	}

	tracker.Reset()
	if tracker.IsOffline("SYSTEM-A") || tracker.IsOffline("SYSTEM-B") {
		t.Errorf("Expected all systems to be online after Reset()")
	}
}

func TestSystemStatusTracker_ConcurrentAccess(t *testing.T) {
	tracker := NewSystemStatusTracker()
	const numGoroutines = 50
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			system := "DEVICE-CONCURRENT"
			for j := 0; j < iterations; j++ {
				tracker.MarkOffline(system)
				_ = tracker.IsOffline(system)
				tracker.MarkOnline(system)
				_ = tracker.IsOffline(system)
			}
		}(i)
	}

	wg.Wait()
}
