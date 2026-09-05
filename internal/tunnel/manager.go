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
// File: internal/tunnel/manager.go
// Author: Gabriel Moraes
// Date: 2026-09-04
// Modified: 2026-09-04 (SOLID Refactor: Pure State Orchestrator & Dependency Injection)

package tunnel

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// Manager orchestrates tunnel lifecycle, state transitions, and auto-reconnections.
// Adheres strictly to Single Responsibility Principle (SRP) by delegating OS/CLI execution to Driver.
// Implements the Service interface (DIP / ISP).
type Manager struct {
	mu           sync.RWMutex
	driver       Driver
	localPort    string
	state        State
	publicURL    string
	domain       string
	authToken    string
	errorMessage string
	startedAt    time.Time

	shouldRun bool
	cancel    context.CancelFunc
}

// NewManager creates an orchestrator with an injected tunnel driver.
func NewManager(driver Driver, localPort string) *Manager {
	if driver == nil {
		driver = NewNgrokDriver()
	}
	if localPort == "" {
		localPort = "8080"
	}
	return &Manager{
		driver:    driver,
		localPort: localPort,
		state:     StateOffline,
	}
}

// IsBinaryAvailable checks if the underlying driver binary is installed.
func (m *Manager) IsBinaryAvailable() bool {
	return m.driver.IsAvailable()
}

// Start launches the tunnel through the injected driver and monitors its state.
func (m *Manager) Start(authToken, domain string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	authToken = strings.TrimSpace(authToken)
	domain = strings.TrimSpace(domain)

	if !m.driver.IsAvailable() {
		m.state = StateError
		m.errorMessage = fmt.Sprintf("Executável do provedor (%s) não foi encontrado no sistema (PATH).", m.driver.Name())
		return fmt.Errorf("driver %s not available", m.driver.Name())
	}

	if authToken == "" {
		m.state = StateError
		m.errorMessage = "Authtoken não configurado."
		return fmt.Errorf("authtoken required")
	}

	// Terminate any running session first
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	_ = m.driver.Stop()

	authToken = CleanAuthToken(authToken)
	domain = CleanDomain(domain)

	m.authToken = authToken
	m.domain = domain
	m.shouldRun = true
	m.state = StateConnecting
	m.errorMessage = ""
	m.publicURL = ""

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	cfg := Config{
		AuthToken: authToken,
		Domain:    domain,
		LocalPort: m.localPort,
	}

	if err := m.driver.Start(ctx, cfg); err != nil {
		m.state = StateError
		m.errorMessage = fmt.Sprintf("Falha ao iniciar túnel: %v", err)
		return err
	}

	m.startedAt = time.Now()
	log.Printf("[TUNNEL] Started via driver '%s'. Polling for readiness...", m.driver.Name())

	// 1. Asynchronously poll driver for public URL
	go m.pollReadiness(ctx, domain)

	// 2. Supervise process exit in background
	go m.supervise(ctx)

	return nil
}

// pollReadiness polls the driver for public URL readiness.
func (m *Manager) pollReadiness(ctx context.Context, fallbackDomain string) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(20 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case <-timeout:
			m.mu.Lock()
			if m.state == StateConnecting {
				if fallbackDomain != "" {
					m.state = StateOnline
					m.publicURL = "https://" + fallbackDomain
					log.Printf("[TUNNEL] Public URL assumed from configured domain: %s", m.publicURL)
				} else {
					m.state = StateError
					m.errorMessage = "Tempo limite excedido aguardando inicialização do túnel."
				}
			}
			m.mu.Unlock()
			return
		case <-ticker.C:
			pollCtx, pollCancel := context.WithTimeout(ctx, 1*time.Second)
			url, err := m.driver.GetPublicURL(pollCtx)
			pollCancel()

			if err == nil && url != "" {
				m.mu.Lock()
				m.state = StateOnline
				m.publicURL = url
				m.errorMessage = ""
				log.Printf("[TUNNEL] Active Public Tunnel: %s", m.publicURL)
				m.mu.Unlock()
				return
			}
		}
	}
}

// supervise monitors process termination and auto-reconnects when appropriate.
func (m *Manager) supervise(ctx context.Context) {
	err := m.driver.Wait()

	if ctx.Err() != nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.shouldRun {
		m.state = StateOffline
		m.publicURL = ""
		return
	}

	log.Printf("[TUNNEL] Process exited unexpectedly: %v", err)
	m.state = StateError
	m.errorMessage = "O processo do túnel foi encerrado inesperadamente."

	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}

		m.mu.RLock()
		shouldRetry := m.shouldRun
		token := m.authToken
		dom := m.domain
		m.mu.RUnlock()

		if shouldRetry && token != "" {
			log.Println("[TUNNEL] Attempting automatic reconnection...")
			_ = m.Start(token, dom)
		}
	}()
}

// Stop cleanly terminates the active tunnel session.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.shouldRun = false
	m.state = StateOffline
	m.publicURL = ""
	m.errorMessage = ""

	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}

	return m.driver.Stop()
}

// GetStatus returns a snapshot of the current tunnel state.
func (m *Manager) GetStatus() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()

	telemetryURL := ""
	if m.publicURL != "" {
		telemetryURL = strings.TrimRight(m.publicURL, "/") + "/api/telemetry"
	}

	started := ""
	if !m.startedAt.IsZero() {
		started = m.startedAt.Format("15:04:05 02/01/2006")
	}

	return Status{
		State:        m.state,
		PublicURL:    m.publicURL,
		TelemetryURL: telemetryURL,
		Domain:       m.domain,
		BinaryFound:  m.driver.IsAvailable(),
		ErrorMessage: m.errorMessage,
		StartedAt:    started,
	}
}
