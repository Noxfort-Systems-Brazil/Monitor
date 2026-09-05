// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: internal/transport/http/database_provision_handler.go
// Author: Gabriel Moraes
// Date: 2026-09-05

package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"noxfort-monitor-server/internal/domain"
	"noxfort-monitor-server/internal/storage"
)

// DatabaseUserProvisioner abstracts user/role provisioning in target database systems.
type DatabaseUserProvisioner interface {
	ProvisionUser(host string, port int, dbname, adminUser, adminPassword, newUser, newPassword, sslmode string) error
}

// PostgresUserProvisioner delegates to storage.ProvisionPostgresUser.
type PostgresUserProvisioner struct{}

// ProvisionUser invokes the storage layer PostgreSQL provisioner.
func (p *PostgresUserProvisioner) ProvisionUser(host string, port int, dbname, adminUser, adminPassword, newUser, newPassword, sslmode string) error {
	return storage.ProvisionPostgresUser(host, port, dbname, adminUser, adminPassword, newUser, newPassword, sslmode)
}

// HandleProvisionUser creates a new database role and schema/database using administrator credentials.
func (h *DatabaseHandler) HandleProvisionUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	req := parseProvisionRequest(r)
	w.Header().Set("Content-Type", "application/json")

	// 1. Provision user via provisioner abstraction
	if err := h.provisioner.ProvisionUser(req.Host, req.Port, req.DBName, req.AdminUser, req.AdminPassword, req.NewUser, req.NewPassword, req.SSLMode); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Falha ao provisionar usuário: %v", err),
		})
		return
	}

	// 2. Automatically switch to the newly created user
	newCfg := domain.DatabaseConfig{
		Type:     "postgres",
		Host:     req.Host,
		Port:     req.Port,
		DBName:   req.DBName,
		Schema:   req.Schema,
		User:     req.NewUser,
		Password: req.NewPassword,
		SSLMode:  req.SSLMode,
	}

	if err := h.dbService.Switch(newCfg, req.Migrate); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Usuário '%s' criado com sucesso, mas falha ao conectar: %v", req.NewUser, err),
		})
		return
	}

	if h.auditRepo != nil {
		details := fmt.Sprintf("Usuário provisionado: %s, Banco: %s, Schema: %s", req.NewUser, req.DBName, req.Schema)
		_ = h.auditRepo.SaveSecurityAuditLog(&domain.SecurityAuditLog{
			Username:  "admin",
			Action:    "POSTGRES_USER_PROVISIONED",
			Details:   details,
			CreatedAt: time.Now(),
		})
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Usuário '%s' criado com sucesso no PostgreSQL e conectado!", req.NewUser),
		"status":  h.dbService.GetStatus(),
	})
}
