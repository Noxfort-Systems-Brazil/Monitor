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
// File: cmd/server/main.go
// Author: Gabriel Moraes
// Date: 2026-01-18

package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	// Internal Packages
	"noxfort-monitor-server/internal/desktop"
	"noxfort-monitor-server/internal/monitor"
	"noxfort-monitor-server/internal/security"
	"noxfort-monitor-server/internal/storage"
	transportHttp "noxfort-monitor-server/internal/transport/http"
	transportMqtt "noxfort-monitor-server/internal/transport/mqtt"
	"noxfort-monitor-server/internal/tunnel"
)

func main() {
	headless := flag.Bool("headless", false, "Run in headless mode without GUI desktop window")
	flag.BoolVar(headless, "server-only", false, "Alias for --headless")
	flag.Parse()

	// 0. Single Instance Lock (Enforce only one monitor open at a time)
	if !*headless {
		if desktop.TryActivateExisting() {
			log.Println("[BOOT] Uma instância do Noxfort Monitor já está em execução. Janela existente trazida para o primeiro plano.")
			os.Exit(0)
		}
	}

	// 1. Initialize Logger
	log.Println("[BOOT] Starting Noxfort Monitor v2.0 (Event-Driven)...")

	// 1.1 Single Instance IPC Server
	var singleInstanceServer io.Closer
	var appRef *desktop.App
	var appRefMu sync.Mutex

	if !*headless {
		srv, err := desktop.StartSingleInstanceServer(func() {
			appRefMu.Lock()
			app := appRef
			appRefMu.Unlock()
			if app != nil {
				app.RestoreWindow()
			}
		})
		if err == nil {
			singleInstanceServer = srv
		} else {
			log.Printf("[WARN] Single instance socket listener could not be started: %v", err)
		}
	}

	// 2. Database Connection (PostgreSQL with auto-schema or SQLite fallback)
	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("[FATAL] Could not resolve home directory: %v", err)
	}
	dataDir := filepath.Join(homedir, "Documentos", "Monitor")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("[FATAL] Could not create data directory %s: %v", dataDir, err)
	}
	dbPath := filepath.Join(dataDir, "monitor_logs.db")
	log.Printf("[BOOT] Data directory: %s", dataDir)

	dbConfig := storage.LoadDatabaseConfig()
	if dbConfig.FilePath == "" {
		dbConfig.FilePath = dbPath
	}

	var db *sql.DB
	var activeDriver string

	if dbConfig.Type == "postgres" {
		log.Printf("[BOOT] Connecting to PostgreSQL at %s:%d (Database: %s, Schema: %s)...",
			dbConfig.Host, dbConfig.Port, dbConfig.DBName, dbConfig.Schema)
		pgDB, drv, pgErr := storage.OpenConnection(dbConfig)
		if pgErr == nil {
			if initErr := storage.InitPostgresSchema(pgDB, dbConfig.Schema); initErr == nil {
				db = pgDB
				activeDriver = drv
				log.Printf("[INFO] PostgreSQL connected successfully (Schema: '%s').", dbConfig.Schema)
			} else {
				log.Printf("[WARN] PostgreSQL connected but failed to initialize schema '%s': %v. Falling back to SQLite.", dbConfig.Schema, initErr)
				_ = pgDB.Close()
			}
		} else {
			log.Printf("[WARN] Failed to connect to PostgreSQL (%v). Falling back to local SQLite.", pgErr)
		}
	}

	// Fallback to SQLite if PostgreSQL was not chosen or failed
	if db == nil {
		sqliteDB, err := storage.NewDatabase(dbPath)
		if err != nil {
			log.Fatalf("[FATAL] Failed to initialize local SQLite database: %v", err)
		}
		db = sqliteDB
		activeDriver = "sqlite"
		dbConfig.Type = "sqlite"
		log.Println("[INFO] SQLite connected (Pure Go Driver).")
	}

	// 2.1 Central DB Manager (supports live hot-reload from Web/Desktop UI)
	dbManager := storage.NewDBManager(db, activeDriver, dbConfig)

	// 3. Initialize Repositories
	deviceRepo := storage.NewDeviceRepository(db)
	contactRepo := storage.NewContactRepository(db)
	settingsRepo := storage.NewSettingsRepository(db)
	telemetryRepo := storage.NewTelemetryRepository(db)
	userRepo := storage.NewUserRepository(db)
	auditRepo := storage.NewAuditRepository(db, activeDriver)

	// Register repositories for live database hot-reloading
	dbManager.RegisterRepository(deviceRepo)
	dbManager.RegisterRepository(contactRepo)
	dbManager.RegisterRepository(settingsRepo)
	dbManager.RegisterRepository(telemetryRepo)
	dbManager.RegisterRepository(userRepo)
	dbManager.RegisterRepository(auditRepo)

	// 4. Initialize Core & Security Services
	emailChan := monitor.NewEmailChannel()
	telegramChan := monitor.NewTelegramChannel()
	channelTester := monitor.NewChannelTester(emailChan, telegramChan)

	alertService := monitor.NewAlertService(contactRepo, settingsRepo, emailChan, telegramChan)
	alertService.SetAuditRepository(auditRepo)

	stateManager := monitor.NewStateManager(telemetryRepo, deviceRepo, alertService)
	secManager := security.NewSecurityManager(userRepo)
	secManager.SetAuditRepository(auditRepo)

	if err := secManager.EnsureSuperuser(); err != nil {
		log.Fatalf("[FATAL] Failed to initialize superuser: %v", err)
	}

	engine := monitor.NewEngine(deviceRepo, telemetryRepo, alertService)
	engine.SetAuditRepository(auditRepo)
	engine.Start()

	// 5. Initialize & Start MQTT Client
	settings, err := settingsRepo.GetSettings()
	if err != nil {
		log.Printf("[WARN] Failed to load settings for MQTT (using defaults): %v", err)
	}
	brokerURL := "tcp://127.0.0.1:1883"
	if settings.MqttAddress != "" {
		brokerURL = settings.MqttAddress
	}
	log.Printf("[BOOT] Connecting to MQTT Broker at %s...", brokerURL)
	mqttClient := transportMqtt.NewClient(brokerURL, stateManager)
	if err := mqttClient.Connect(); err != nil {
		log.Fatalf("[FATAL] Could not connect to MQTT Broker: %v", err)
	}
	log.Println("[INFO] MQTT Listener Active (Listening for JSON events).")

	// 6. Initialize HTTP Server & Tunnel Manager (Ngrok)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	tunnelDriver := tunnel.NewNgrokDriver()
	tunnelManager := tunnel.NewManager(tunnelDriver, port)
	if settings.NgrokAuthToken != "" {
		log.Printf("[BOOT] Auto-starting Ngrok Tunnel on domain '%s'...", settings.NgrokDomain)
		if err := tunnelManager.Start(settings.NgrokAuthToken, settings.NgrokDomain); err != nil {
			log.Printf("[WARN] Failed to auto-start Ngrok tunnel on boot: %v", err)
		}
	}

	httpServer := transportHttp.NewServer(
		":"+port,
		deviceRepo,
		telemetryRepo,
		contactRepo,
		settingsRepo,
		stateManager,
		channelTester,
		secManager,
		tunnelManager,
		dbManager,
		auditRepo,
	)

	localIP := transportHttp.GetLocalIP()
	serverURL := fmt.Sprintf("http://%s:%s", localIP, port)

	// Run HTTP server in background for external IoT ingestion (/api/telemetry)
	// and Wails native webview
	go func() {
		log.Printf("[INFO] Local REST & Telemetry Server running at %s (Local: http://localhost:%s)", serverURL, port)
		if err := httpServer.Run(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[FATAL] Web Server failed: %v", err)
		}
	}()

	// 7. Graceful Shutdown Coordinator (guaranteed to execute at most once)
	var shutdownOnce sync.Once
	shutdown := func() {
		shutdownOnce.Do(func() {
			log.Println("[SHUTDOWN] Stopping Noxfort Monitor...")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if singleInstanceServer != nil {
				_ = singleInstanceServer.Close()
			}
			if err := tunnelManager.Stop(); err != nil {
				log.Printf("[WARN] Tunnel shutdown error: %v", err)
			}
			if err := httpServer.Stop(shutdownCtx); err != nil {
				log.Printf("[WARN] HTTP Server shutdown error: %v", err)
			}
			engine.Stop()
			mqttClient.Disconnect()
			if activeDB := dbManager.GetDB(); activeDB != nil {
				_ = activeDB.Close()
			}
			log.Println("[SHUTDOWN] All services stopped cleanly.")
		})
	}
	defer shutdown()

	// 8. Run in Headless or Desktop Mode
	if *headless {
		log.Println("[BOOT] Running in HEADLESS mode (no GUI window). Press Ctrl+C to stop.")
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		shutdown()
		os.Exit(0)
	} else {
		log.Println("[BOOT] Starting Noxfort Monitor Desktop GUI (Wails v2)...")
		desktopApp := desktop.New(httpServer.Handler(), shutdown)
		appRefMu.Lock()
		appRef = desktopApp
		appRefMu.Unlock()

		if err := desktopApp.Run(); err != nil {
			log.Fatalf("[FATAL] Desktop application failed: %v", err)
		}
		shutdown()
		os.Exit(0)
	}
}
