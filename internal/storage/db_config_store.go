// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: internal/storage/db_config_store.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package storage

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"noxfort-monitor-server/internal/domain"
)

// DefaultDatabaseConfig returns the default database configuration (fallback to SQLite, configured for PostgreSQL integration).
func DefaultDatabaseConfig() domain.DatabaseConfig {
	homedir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homedir, "Documentos", "Monitor")
	return domain.DatabaseConfig{
		Type:     "postgres",
		Host:     "localhost",
		Port:     5432,
		User:     "user_synapse", // or postgres / user_carina
		Password: "",
		DBName:   "banco_de_dados_noxfort",
		Schema:   "schema_monitor",
		SSLMode:  "disable",
		FilePath: filepath.Join(dataDir, "monitor_logs.db"),
	}
}

// GetDBConfigFilepath returns the path to the database configuration JSON file.
func GetDBConfigFilepath() string {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return "database_config.json"
	}
	dataDir := filepath.Join(homedir, "Documentos", "Monitor")
	_ = os.MkdirAll(dataDir, 0755)
	return filepath.Join(dataDir, "database_config.json")
}

// LoadDatabaseConfig loads database configuration from JSON file, applying environment variable overrides.
func LoadDatabaseConfig() domain.DatabaseConfig {
	cfg := DefaultDatabaseConfig()
	configFile := GetDBConfigFilepath()

	if data, err := os.ReadFile(configFile); err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			log.Printf("[STORAGE] Warning: failed to parse %s: %v", configFile, err)
		}
	}

	// Environment variable overrides
	if val := os.Getenv("MONITOR_DB_TYPE"); val != "" {
		cfg.Type = strings.ToLower(val)
	}
	if val := os.Getenv("MONITOR_DB_HOST"); val != "" {
		cfg.Host = val
	}
	if val := os.Getenv("MONITOR_DB_PORT"); val != "" {
		if p, err := strconv.Atoi(val); err == nil {
			cfg.Port = p
		}
	}
	if val := os.Getenv("MONITOR_DB_USER"); val != "" {
		cfg.User = val
	}
	if val := os.Getenv("MONITOR_DB_PASSWORD"); val != "" {
		cfg.Password = val
	}
	if val := os.Getenv("MONITOR_DB_NAME"); val != "" {
		cfg.DBName = val
	}
	if val := os.Getenv("MONITOR_DB_SCHEMA"); val != "" {
		cfg.Schema = val
	}
	if val := os.Getenv("MONITOR_DB_SSLMODE"); val != "" {
		cfg.SSLMode = val
	}

	// Normalizations
	if cfg.Type == "" {
		cfg.Type = "sqlite"
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

	return cfg
}

// SaveDatabaseConfig writes the database configuration to the persistent JSON file.
func SaveDatabaseConfig(cfg domain.DatabaseConfig) error {
	configFile := GetDBConfigFilepath()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0600)
}
