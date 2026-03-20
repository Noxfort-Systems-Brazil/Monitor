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
	"log"
	"os"
	"path/filepath"

	// Internal Packages
	"noxfort-monitor-server/internal/monitor"
	"noxfort-monitor-server/internal/storage"
	transportHttp "noxfort-monitor-server/internal/transport/http"
	transportMqtt "noxfort-monitor-server/internal/transport/mqtt"
	"noxfort-monitor-server/internal/tray"
)

func main() {
	// 1. Initialize Logger
	log.Println("[BOOT] Starting Noxfort Monitor Server v2.0 (Event-Driven)...")

	// 2. Database Connection
	// The database lives in ~/Documentos/Monitor/ to separate user data from
	// the application source code (XDG best-practice for Linux desktop apps).
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
	db, err := storage.NewDatabase(dbPath)
	if err != nil {
		log.Fatalf("[FATAL] Failed to initialize database: %v", err)
	}
	defer db.Close()
	log.Println("[INFO] Database connected (Pure Go Driver).")

	// 3. Initialize Repositories
	deviceRepo := storage.NewDeviceRepository(db)
	contactRepo := storage.NewContactRepository(db)
	settingsRepo := storage.NewSettingsRepository(db)
	telemetryRepo := storage.NewTelemetryRepository(db)

	// 4. Initialize Core Services
	alertService := monitor.NewAlertService(contactRepo, settingsRepo)
	stateManager := monitor.NewStateManager(telemetryRepo, deviceRepo, alertService)

	engine := monitor.NewEngine(deviceRepo, telemetryRepo, alertService)
	engine.Start()
	defer engine.Stop()

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
	defer mqttClient.Disconnect()
	log.Println("[INFO] MQTT Listener Active (Listening for JSON events).")

	// 6. Initialize HTTP Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	httpServer := transportHttp.NewServer(
		":"+port,
		deviceRepo,
		telemetryRepo,
		contactRepo,
		settingsRepo,
		stateManager,
		alertService,
	)

	// 7. Run HTTP server in a background goroutine so the main goroutine
	//    remains available for the system tray (required by Linux/GTK).
	go func() {
		log.Printf("[INFO] Web Interface running at http://localhost:%s", port)
		log.Println("[INFO] Ready.")
		if err := httpServer.Run(); err != nil {
			log.Fatalf("[FATAL] Web Server failed: %v", err)
		}
	}()

	// 8. System Tray — blocks the main goroutine until the user exits.
	//    On exit: cleanly shut everything down.
	log.Println("[TRAY] Starting system tray icon...")
	tray.Start(port, func() {
		log.Println("[TRAY] Shutting down...")
		engine.Stop()
		mqttClient.Disconnect()
		db.Close()
		os.Exit(0)
	})
}
