// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: internal/storage/dialect.go
// Author: Gabriel Moraes
// Date: 2026-09-05

package storage

import (
	"database/sql"
	"fmt"
	"sync"

	"noxfort-monitor-server/internal/domain"
)

// Dialect encapsulates driver-specific database operations and SQL dialect variances.
type Dialect interface {
	DriverName() string
	Open(cfg domain.DatabaseConfig) (*sql.DB, error)
	GetVersion(db *sql.DB) (string, error)
	CheckSchemaExists(db *sql.DB, schema string) (bool, error)
	InitSchema(db *sql.DB, schema string) error
}

var (
	dialectsMu sync.RWMutex
	dialects   = make(map[string]Dialect)
)

// RegisterDialect registers a database dialect implementation.
func RegisterDialect(name string, d Dialect) {
	dialectsMu.Lock()
	defer dialectsMu.Unlock()
	dialects[name] = d
}

// GetDialect retrieves the dialect by driver name. Defaults to sqlite if empty or unrecognized.
func GetDialect(name string) (Dialect, error) {
	dialectsMu.RLock()
	defer dialectsMu.RUnlock()

	if name == "" {
		name = "sqlite"
	}

	d, ok := dialects[name]
	if !ok {
		if fallback, exists := dialects["sqlite"]; exists {
			return fallback, nil
		}
		return nil, fmt.Errorf("unsupported database dialect: %s", name)
	}
	return d, nil
}

// OpenConnection opens a database connection based on the supplied DatabaseConfig without modifying manager state.
func OpenConnection(cfg domain.DatabaseConfig) (*sql.DB, string, error) {
	d, err := GetDialect(cfg.Type)
	if err != nil {
		return nil, "", err
	}

	db, err := d.Open(cfg)
	if err != nil {
		return nil, "", err
	}

	return db, d.DriverName(), nil
}
