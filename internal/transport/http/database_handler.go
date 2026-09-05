// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: internal/transport/http/database_handler.go
// Author: Gabriel Moraes
// Date: 2026-09-04
// Modified: 2026-09-05 (SOLID Refactor)

package http

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"noxfort-monitor-server/internal/appdir"
	"noxfort-monitor-server/internal/domain"
)

// DatabaseService defines the operations required by DatabaseHandler to monitor, test, and switch storage engines.
type DatabaseService interface {
	GetStatus() domain.DatabaseStatus
	GetConfig() domain.DatabaseConfig
	TestConnection(cfg domain.DatabaseConfig) (domain.DatabaseStatus, error)
	Switch(newCfg domain.DatabaseConfig, migrate bool) error
}

// DatabaseHandler manages database connection settings, testing, and schema provisioning APIs.
type DatabaseHandler struct {
	dbService   DatabaseService
	auditRepo   domain.AuditRepository
	provisioner DatabaseUserProvisioner
}

// NewDatabaseHandler initializes the DatabaseHandler with the default PostgreSQL provisioner.
func NewDatabaseHandler(dbService DatabaseService, auditRepo domain.AuditRepository) *DatabaseHandler {
	return NewDatabaseHandlerWithProvisioner(dbService, auditRepo, &PostgresUserProvisioner{})
}

// NewDatabaseHandlerWithProvisioner initializes the DatabaseHandler with a custom user provisioner for testability.
func NewDatabaseHandlerWithProvisioner(dbService DatabaseService, auditRepo domain.AuditRepository, provisioner DatabaseUserProvisioner) *DatabaseHandler {
	return &DatabaseHandler{
		dbService:   dbService,
		auditRepo:   auditRepo,
		provisioner: provisioner,
	}
}

// ServePage renders the dedicated Server & Database configuration page.
func (h *DatabaseHandler) ServePage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles(
		appdir.Path("web/templates/layout.html"),
		appdir.Path("web/templates/server.html"),
	)
	if err != nil {
		log.Printf("[SERVER_PAGE] Template error: %v", err)
		http.Error(w, "Template Error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title": "Server & Storage Engine",
	}

	_ = tmpl.Execute(w, data)
}

// HandleStatus returns the live status and active configuration of the database.
func (h *DatabaseHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	status := h.dbService.GetStatus()
	cfg := h.dbService.GetConfig()

	resp := map[string]interface{}{
		"status": status,
		"config": map[string]interface{}{
			"type":      cfg.Type,
			"host":      cfg.Host,
			"port":      cfg.Port,
			"user":      cfg.User,
			"dbname":    cfg.DBName,
			"schema":    cfg.Schema,
			"sslmode":   cfg.SSLMode,
			"file_path": cfg.FilePath,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// HandleTest evaluates connectivity against supplied PostgreSQL or SQLite credentials.
func (h *DatabaseHandler) HandleTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg, _ := parseDatabaseConfig(r)
	status, err := h.dbService.TestConnection(cfg)

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
			"status":  status,
		})
		return
	}

	message := formatConnectionTestMessage(status, cfg)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": message,
		"status":  status,
	})
}

// HandleSave applies new database settings, creates the schema if needed, and hot-reloads the connection.
func (h *DatabaseHandler) HandleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg, migrate := parseDatabaseConfig(r)

	if err := h.dbService.Switch(cfg, migrate); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Falha ao inicializar banco: %v", err),
		})
		return
	}

	if h.auditRepo != nil {
		details := fmt.Sprintf("Motor: %s, Banco: %s, Schema: %s, Migração: %v", cfg.Type, cfg.DBName, cfg.Schema, migrate)
		_ = h.auditRepo.SaveSecurityAuditLog(&domain.SecurityAuditLog{
			Username:  "admin",
			Action:    "DATABASE_CONFIG_UPDATED",
			Details:   details,
			CreatedAt: time.Now(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Banco de dados configurado com sucesso no schema '%s'!", cfg.Schema),
		"status":  h.dbService.GetStatus(),
	})
}
