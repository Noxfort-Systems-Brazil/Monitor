// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: internal/transport/http/database_dto.go
// Author: Gabriel Moraes
// Date: 2026-09-05

package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"noxfort-monitor-server/internal/domain"
)

// parseDatabaseConfig extracts DatabaseConfig from JSON body or URL-encoded form.
func parseDatabaseConfig(r *http.Request) (domain.DatabaseConfig, bool) {
	var cfg domain.DatabaseConfig
	var migrate bool

	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		var req struct {
			domain.DatabaseConfig
			Migrate bool `json:"migrate"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		cfg = req.DatabaseConfig
		migrate = req.Migrate
	} else {
		_ = r.ParseForm()
		port, _ := strconv.Atoi(r.FormValue("port"))
		if port == 0 {
			port = 5432
		}
		cfg = domain.DatabaseConfig{
			Type:     strings.TrimSpace(r.FormValue("type")),
			Host:     strings.TrimSpace(r.FormValue("host")),
			Port:     port,
			User:     strings.TrimSpace(r.FormValue("user")),
			Password: r.FormValue("password"),
			DBName:   strings.TrimSpace(r.FormValue("dbname")),
			Schema:   strings.TrimSpace(r.FormValue("schema")),
			SSLMode:  strings.TrimSpace(r.FormValue("sslmode")),
			FilePath: strings.TrimSpace(r.FormValue("file_path")),
		}
		migrate = r.FormValue("migrate") == "true" || r.FormValue("migrate") == "on" || r.FormValue("migrate") == "1"
	}

	if cfg.Type == "" {
		cfg.Type = "postgres"
	}
	if cfg.Schema == "" {
		cfg.Schema = "schema_monitor"
	}
	if cfg.Port == 0 {
		cfg.Port = 5432
	}
	if cfg.SSLMode == "" {
		cfg.SSLMode = "disable"
	}

	return cfg, migrate
}

type provisionRequest struct {
	Host          string `json:"host"`
	Port          int    `json:"port"`
	DBName        string `json:"dbname"`
	Schema        string `json:"schema"`
	AdminUser     string `json:"admin_user"`
	AdminPassword string `json:"admin_password"`
	NewUser       string `json:"new_user"`
	NewPassword   string `json:"new_password"`
	SSLMode       string `json:"sslmode"`
	Migrate       bool   `json:"migrate"`
}

func parseProvisionRequest(r *http.Request) provisionRequest {
	var req provisionRequest

	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		_ = json.NewDecoder(r.Body).Decode(&req)
	} else {
		_ = r.ParseForm()
		port, _ := strconv.Atoi(r.FormValue("port"))
		req.Host = r.FormValue("host")
		req.Port = port
		req.DBName = r.FormValue("dbname")
		req.Schema = r.FormValue("schema")
		req.AdminUser = r.FormValue("admin_user")
		req.AdminPassword = r.FormValue("admin_password")
		req.NewUser = r.FormValue("new_user")
		req.NewPassword = r.FormValue("new_password")
		req.SSLMode = r.FormValue("sslmode")
		req.Migrate = r.FormValue("migrate") == "true" || r.FormValue("migrate") == "on" || r.FormValue("migrate") == "1"
	}

	if req.AdminUser == "" {
		req.AdminUser = "postgres"
	}
	if req.NewUser == "" {
		req.NewUser = "user_monitor"
	}
	if req.Port == 0 {
		req.Port = 5432
	}
	if req.DBName == "" {
		req.DBName = "banco_de_dados_noxfort"
	}
	if req.Schema == "" {
		req.Schema = "schema_monitor"
	}

	return req
}

// formatConnectionTestMessage generates a user-facing success feedback message based on schema detection.
func formatConnectionTestMessage(status domain.DatabaseStatus, cfg domain.DatabaseConfig) string {
	message := fmt.Sprintf("Conexão bem-sucedida ao banco '%s'!", cfg.DBName)
	if cfg.Type != "sqlite" && cfg.Schema != "" {
		if status.SchemaExists {
			message = fmt.Sprintf("Conexão bem-sucedida! Schema '%s' já existe e está pronto.", cfg.Schema)
		} else {
			message = fmt.Sprintf("Conexão bem-sucedida! Schema '%s' será criado automaticamente ao salvar.", cfg.Schema)
		}
	}
	return message
}
