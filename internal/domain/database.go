// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: internal/domain/database.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package domain

import "time"

// DatabaseConfig holds connection settings for the persistence layer.
type DatabaseConfig struct {
	Type     string `json:"type"`      // "postgres" or "sqlite"
	Host     string `json:"host"`      // e.g. "localhost"
	Port     int    `json:"port"`      // e.g. 5432
	User     string `json:"user"`      // e.g. "user_monitor", "user_carina", "postgres"
	Password string `json:"password"`  // e.g. "database123"
	DBName   string `json:"dbname"`    // e.g. "banco_de_dados_noxfort"
	Schema   string `json:"schema"`    // e.g. "schema_monitor"
	SSLMode  string `json:"sslmode"`   // e.g. "disable", "require"
	FilePath string `json:"file_path"` // SQLite file path e.g. "monitor_logs.db"
}

// DatabaseStatus represents the real-time health and metadata of the active database connection.
type DatabaseStatus struct {
	Connected    bool      `json:"connected"`
	Type         string    `json:"type"`          // "postgres" or "sqlite"
	Host         string    `json:"host"`
	Port         int       `json:"port"`
	DBName       string    `json:"dbname"`
	Schema       string    `json:"schema"`
	User         string    `json:"user"`
	LatencyMs    int64     `json:"latency_ms"`
	SchemaExists bool      `json:"schema_exists"`
	ServerTime   time.Time `json:"server_time"`
	Version      string    `json:"version"`
	ErrorMessage string    `json:"error_message,omitempty"`
}
