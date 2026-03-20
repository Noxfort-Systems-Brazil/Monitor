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
// File: internal/monitor/engine.go
// Author: Gabriel Moraes
// Date: 2026-01-19

package monitor

import (
	"fmt"
	"log"
	"sync"
	"time"

	"noxfort-monitor-server/internal/domain"
)

// Engine (Watchdog) is responsible for detecting silent failures.
// It checks periodically if monitored systems have stopped sending signals.
type Engine struct {
	deviceRepo    domain.DeviceRepository
	telemetryRepo domain.TelemetryRepository
	alertService  *AlertService

	// OfflineStatus tracks the state in memory to avoid spamming alerts.
	// Map: Identifier (string) -> IsOffline (bool)
	offlineStatus map[string]bool
	mu            sync.RWMutex

	ticker   *time.Ticker
	stopChan chan bool
}

// NewEngine creates the Watchdog worker.
func NewEngine(dRepo domain.DeviceRepository, tRepo domain.TelemetryRepository, alerts *AlertService) *Engine {
	return &Engine{
		deviceRepo:    dRepo,
		telemetryRepo: tRepo,
		alertService:  alerts,
		offlineStatus: make(map[string]bool),
		stopChan:      make(chan bool),
	}
}

// Start begins the monitoring loop in a background goroutine.
func (e *Engine) Start() {
	e.ticker = time.NewTicker(30 * time.Second) // Check every 30s
	e.stopChan = make(chan bool)

	go func() {
		log.Println("[ENGINE] Watchdog started. Monitoring system heartbeats...")
		for {
			select {
			case <-e.stopChan:
				return
			case <-e.ticker.C:
				e.checkSystems()
			}
		}
	}()
}

// Stop halts the monitoring loop.
func (e *Engine) Stop() {
	if e.ticker != nil {
		e.ticker.Stop()
	}
	if e.stopChan != nil {
		e.stopChan <- true
	}
	log.Println("[ENGINE] Watchdog stopped.")
}

// checkSystems iterates over all enabled systems to verify their LastSeen timestamp.
func (e *Engine) checkSystems() {
	devices, err := e.deviceRepo.GetAllDevices()
	if err != nil {
		log.Printf("[ENGINE] Failed to query devices: %v", err)
		return
	}

	threshold := 5 * time.Minute // TODO: Could be configurable per system

	for _, dev := range devices {
		if !dev.Enabled {
			continue
		}

		timeSince := time.Since(dev.LastSeen)

		e.mu.Lock()
		isKnownOffline := e.offlineStatus[dev.Identifier]

		if timeSince > threshold {
			// CASE 1: System is OFFLINE (Timeout)
			if !isKnownOffline {
				log.Printf("[ENGINE] System %s went OFFLINE (Last seen: %v)", dev.Identifier, timeSince)
				e.offlineStatus[dev.Identifier] = true // Mark as offline

				// Generate internal event
				e.triggerEvent(dev.Identifier, domain.LevelCritical, fmt.Sprintf("System OFFLINE: No signal for %v.", timeSince.Round(time.Second)))
			}
		} else {
			// CASE 2: System is ONLINE
			if isKnownOffline {
				log.Printf("[ENGINE] System %s recovered (ONLINE)", dev.Identifier)
				delete(e.offlineStatus, dev.Identifier) // Remove from offline map

				// Generate internal recovery event
				e.triggerEvent(dev.Identifier, domain.LevelInfo, "System ONLINE: Signal recovered.")
			}
		}
		e.mu.Unlock()
	}
}

// triggerEvent creates a synthetic event and injects it into the alert pipeline.
func (e *Engine) triggerEvent(identifier string, level domain.EventLevel, msg string) {
	event := &domain.IncomingEvent{
		Origin:     "monitor-watchdog",
		Level:      level,
		Message:    msg,
		OccurredAt: time.Now(),
	}

	// 1. Persist to DB
	if err := e.telemetryRepo.SaveEvent(identifier, event); err != nil {
		log.Printf("[ENGINE] Failed to save watchdog event: %v", err)
	}

	// 2. Alert Human
	e.alertService.TriggerAlert(identifier, event)
}
