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
// Modified: 2026-09-04 (SOLID Refactor)

package monitor

import (
	"fmt"
	"log"
	"sync"
	"time"

	"noxfort-monitor-server/internal/domain"
)

// DeviceLister retrieves registered devices to evaluate their health.
type DeviceLister interface {
	GetAllDevices() ([]domain.Device, error)
}

// EventRecorder persists generated watchdog health events.
type EventRecorder interface {
	SaveEvent(identifier string, event *domain.IncomingEvent) error
}

// EngineConfig controls watchdog timing and timeouts.
type EngineConfig struct {
	CheckInterval    time.Duration
	OfflineThreshold time.Duration
}

// DefaultEngineConfig provides standard production defaults (30s interval, 5m threshold).
func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		CheckInterval:    30 * time.Second,
		OfflineThreshold: 5 * time.Minute,
	}
}

// Engine (Watchdog) is responsible for detecting silent failures.
// It checks periodically if monitored systems have stopped sending signals.
type Engine struct {
	devices   DeviceLister
	events    EventRecorder
	alerts    AlertDispatcher
	auditRepo domain.AuditRepository
	tracker   *SystemStatusTracker
	config    EngineConfig

	ticker   *time.Ticker
	stopChan chan struct{}
	stopOnce sync.Once
}

// SetAuditRepository attaches the audit repository for recording device availability transitions.
func (e *Engine) SetAuditRepository(repo domain.AuditRepository) {
	e.auditRepo = repo
}

// NewEngine creates the Watchdog worker with production default intervals.
func NewEngine(devices DeviceLister, events EventRecorder, alerts AlertDispatcher) *Engine {
	return NewEngineWithConfig(devices, events, alerts, DefaultEngineConfig())
}

// NewEngineWithConfig creates the Watchdog worker with custom timing configurations (useful for tests).
func NewEngineWithConfig(
	devices DeviceLister,
	events EventRecorder,
	alerts AlertDispatcher,
	config EngineConfig,
) *Engine {
	if config.CheckInterval <= 0 {
		config.CheckInterval = 30 * time.Second
	}
	if config.OfflineThreshold <= 0 {
		config.OfflineThreshold = 5 * time.Minute
	}

	return &Engine{
		devices:  devices,
		events:   events,
		alerts:   alerts,
		tracker:  NewSystemStatusTracker(),
		config:   config,
		stopChan: make(chan struct{}),
	}
}

// Start begins the monitoring loop in a background goroutine.
func (e *Engine) Start() {
	e.ticker = time.NewTicker(e.config.CheckInterval)
	e.stopChan = make(chan struct{})
	e.stopOnce = sync.Once{}

	go func() {
		log.Println("[ENGINE] Watchdog started. Monitoring system heartbeats...")
		for {
			select {
			case <-e.stopChan:
				return
			case <-e.ticker.C:
				e.CheckSystemsOnce()
			}
		}
	}()
}

// Stop halts the monitoring loop. It is safe and idempotent to call multiple times.
func (e *Engine) Stop() {
	e.stopOnce.Do(func() {
		if e.ticker != nil {
			e.ticker.Stop()
		}
		if e.stopChan != nil {
			close(e.stopChan)
		}
		log.Println("[ENGINE] Watchdog stopped.")
	})
}

// CheckSystemsOnce iterates over all enabled systems to verify their LastSeen timestamp.
// Exposing this enables deterministic unit testing without artificial sleeps.
func (e *Engine) CheckSystemsOnce() {
	if e.devices == nil {
		return
	}

	devices, err := e.devices.GetAllDevices()
	if err != nil {
		log.Printf("[ENGINE] Failed to query devices: %v", err)
		return
	}

	for _, dev := range devices {
		if !dev.Enabled {
			continue
		}

		timeSince := time.Since(dev.LastSeen)

		if timeSince > e.config.OfflineThreshold {
			// CASE 1: System transitioned to OFFLINE
			if e.tracker.MarkOffline(dev.Identifier) {
				log.Printf("[ENGINE] System %s went OFFLINE (Last seen: %v)", dev.Identifier, timeSince)
				e.triggerEvent(dev.Identifier, domain.LevelCritical, fmt.Sprintf("System OFFLINE: No signal for %v.", timeSince.Round(time.Second)))
				if e.auditRepo != nil {
					_ = e.auditRepo.SaveDeviceStateTransition(&domain.DeviceStateTransition{
						DeviceIdentifier:   dev.Identifier,
						PreviousState:      "ONLINE",
						NewState:           "OFFLINE",
						DurationOfflineSec: 0,
						TransitionAt:       time.Now(),
					})
				}
			}
		} else {
			// CASE 2: System recovered to ONLINE
			if e.tracker.MarkOnline(dev.Identifier) {
				log.Printf("[ENGINE] System %s recovered (ONLINE)", dev.Identifier)
				e.triggerEvent(dev.Identifier, domain.LevelInfo, "System ONLINE: Signal recovered.")
				if e.auditRepo != nil {
					_ = e.auditRepo.SaveDeviceStateTransition(&domain.DeviceStateTransition{
						DeviceIdentifier:   dev.Identifier,
						PreviousState:      "OFFLINE",
						NewState:           "ONLINE",
						DurationOfflineSec: int64(timeSince.Seconds()),
						TransitionAt:       time.Now(),
					})
				}
			}
		}
	}
}

// triggerEvent creates a synthetic event and injects it into the alert pipeline.
func (e *Engine) triggerEvent(identifier string, level domain.EventLevel, msg string) {
	event := &domain.IncomingEvent{
		Category:   domain.CategoryHardware,
		Origin:     "monitor-watchdog",
		Level:      level,
		Message:    msg,
		OccurredAt: time.Now(),
	}

	// 1. Persist to DB if event recorder is provided
	if e.events != nil {
		if err := e.events.SaveEvent(identifier, event); err != nil {
			log.Printf("[ENGINE] Failed to save watchdog event: %v", err)
		}
	}

	// 2. Alert Human if dispatcher is provided
	if e.alerts != nil {
		e.alerts.TriggerAlert(identifier, event)
	}
}
