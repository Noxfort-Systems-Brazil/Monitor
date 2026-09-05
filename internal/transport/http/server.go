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
// File: internal/transport/http/server.go
// Author: Gabriel Moraes
// Date: 2026-01-19
// Modified: 2026-09-04 (SOLID Refactor)

package http

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"noxfort-monitor-server/internal/appdir"
	"noxfort-monitor-server/internal/domain"
	"noxfort-monitor-server/internal/monitor"
	"noxfort-monitor-server/internal/tunnel"
)

// SecurityService abstracts authentication, user management, and session validation (ISP / DIP).
type SecurityService interface {
	AuthService
	UserManagementService
	SessionValidator
}

// Server is the HTTP router and service lifecycle orchestrator.
type Server struct {
	addr string

	// Modular Handlers
	dashboardHandler *DashboardHandler
	deviceHandler    *DeviceHandler
	contactHandler   *ContactHandler
	settingsHandler  *SettingsHandler
	databaseHandler  *DatabaseHandler
	auditHandler     *AuditHandler
	authHandler      *AuthHandler
	userHandler      *UserHandler
	telemetryHandler *TelemetryHandler
	browserHandler   *BrowserHandler
	tunnelHandler    *TunnelHandler

	authMiddleware *AuthMiddleware

	httpServer *http.Server
}

// NewServer initializes the HTTP server, assembling modular sub-handlers and middleware.
// Strictly adheres to Dependency Inversion Principle (DIP) by depending solely on interfaces.
func NewServer(
	addr string,
	dRepo domain.DeviceRepository,
	tRepo domain.TelemetryRepository,
	cRepo domain.ContactRepository,
	sRepo domain.SettingsRepository,
	sm monitor.EventProcessor,
	tester ConnectionTester,
	secService SecurityService,
	tm tunnel.Service,
	dbService DatabaseService,
	auditRepo domain.AuditRepository,
) *Server {
	var authHandler *AuthHandler
	var userHandler *UserHandler
	if secService != nil {
		authHandler = NewAuthHandler(secService)
		userHandler = NewUserHandler(secService, secService)
	}

	var dbH *DatabaseHandler
	if dbService != nil {
		dbH = NewDatabaseHandler(dbService, auditRepo)
	}

	var auditH *AuditHandler
	if auditRepo != nil {
		auditH = NewAuditHandler(auditRepo)
	}

	return &Server{
		addr:             addr,
		dashboardHandler: NewDashboardHandler(dRepo, tRepo),
		deviceHandler:    NewDeviceHandler(dRepo, tm),
		contactHandler:   NewContactHandler(cRepo),
		settingsHandler:  NewSettingsHandler(sRepo, tester),
		databaseHandler:  dbH,
		auditHandler:     auditH,
		authHandler:      authHandler,
		userHandler:      userHandler,
		telemetryHandler: NewTelemetryHandler(sm),
		browserHandler:   NewBrowserHandler(),
		tunnelHandler:    NewTunnelHandler(sRepo, tm),
		authMiddleware:   NewAuthMiddleware(authHandler),
	}
}

// Handler configures the HTTP ServeMux and applies the security middleware.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// 1. Static Assets (Public)
	fs := http.FileServer(http.Dir(appdir.Path("web/static")))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// 2. Authentication Pages & APIs
	mux.HandleFunc("/login", s.authHandler.ServeLogin)
	mux.HandleFunc("/register", s.authHandler.ServeRegister)
	mux.HandleFunc("/api/auth/login", s.authHandler.HandleLogin)
	mux.HandleFunc("/api/auth/register", s.userHandler.HandleRegister)
	mux.HandleFunc("/api/auth/logout", s.authHandler.HandleLogout)
	mux.HandleFunc("/api/auth/status", s.authHandler.HandleStatus)

	// 3. IoT Telemetry API (HTTP POST Ingest)
	mux.HandleFunc("/api/telemetry", s.telemetryHandler.HandleIngest)

	// 4. Protected Application Routes
	mux.HandleFunc("/", s.dashboardHandler.ServePage)

	// System Management
	mux.HandleFunc("/devices", s.deviceHandler.ServePage)
	mux.HandleFunc("/devices/delete", s.deviceHandler.HandleDelete)

	// Response Team (Contacts)
	mux.HandleFunc("/contacts", s.contactHandler.ServePage)
	mux.HandleFunc("/contacts/create", s.contactHandler.HandleCreate)
	mux.HandleFunc("/contacts/update", s.contactHandler.HandleUpdate)
	mux.HandleFunc("/contacts/delete", s.contactHandler.HandleDelete)

	// Remote Access & Ingestion Tunnel (Ngrok)
	mux.HandleFunc("/remote", s.tunnelHandler.ServePage)
	mux.HandleFunc("/api/tunnel/status", s.tunnelHandler.HandleStatus)
	mux.HandleFunc("/api/tunnel/save", s.tunnelHandler.HandleSave)
	mux.HandleFunc("/api/tunnel/start", s.tunnelHandler.HandleStart)
	mux.HandleFunc("/api/tunnel/stop", s.tunnelHandler.HandleStop)

	// Settings
	mux.HandleFunc("/settings", s.settingsHandler.ServePage)
	mux.HandleFunc("/settings/save", s.settingsHandler.HandleSave)
	mux.HandleFunc("/settings/test", s.settingsHandler.HandleTest)
	mux.HandleFunc("/settings/test-telegram", s.settingsHandler.HandleTestTelegram)

	// Database Management & Server Configuration
	if s.databaseHandler != nil {
		mux.HandleFunc("/server", s.databaseHandler.ServePage)
		mux.HandleFunc("/api/settings/database/status", s.databaseHandler.HandleStatus)
		mux.HandleFunc("/api/settings/database/test", s.databaseHandler.HandleTest)
		mux.HandleFunc("/api/settings/database/save", s.databaseHandler.HandleSave)
		mux.HandleFunc("/api/settings/database/provision-user", s.databaseHandler.HandleProvisionUser)
	}

	// Audit Trail
	if s.auditHandler != nil {
		mux.HandleFunc("/audit", s.auditHandler.ServePage)
		mux.HandleFunc("/api/audit/security", s.auditHandler.HandleSecurityLogs)
		mux.HandleFunc("/api/audit/alerts", s.auditHandler.HandleAlertLogs)
		mux.HandleFunc("/api/audit/transitions", s.auditHandler.HandleTransitionLogs)
	}

	// Account Management
	mux.HandleFunc("/users", s.userHandler.ServePage)
	mux.HandleFunc("/api/users", s.userHandler.HandleList)
	mux.HandleFunc("/api/users/create", s.userHandler.HandleCreateUser)
	mux.HandleFunc("/api/users/delete", s.userHandler.HandleDelete)

	// 5. Open External Links in Default Browser
	mux.HandleFunc("/api/open-external", s.browserHandler.HandleOpenExternal)

	// 6. Window Controls (Fallback for standalone browser/headless mode)
	mux.HandleFunc("/api/window/toggle-fullscreen", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"desktop": false,
		})
	})
	mux.HandleFunc("/api/window/exit-fullscreen", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"desktop": false,
		})
	})

	// Wrap routing tree with Auth and RBAC middleware
	return s.authMiddleware.Wrap(mux)
}

// Run configures the routes and starts the listening loop.
func (s *Server) Run() error {
	handler := s.Handler()

	s.httpServer = &http.Server{
		Addr:         s.addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Printf("🌍 HTTP Server listening on %s", s.addr)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully shuts down the HTTP server.
func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}
