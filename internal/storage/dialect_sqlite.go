// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: internal/storage/dialect_sqlite.go
// Author: Gabriel Moraes
// Date: 2026-09-05

package storage

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // Pure Go SQLite Driver
	"noxfort-monitor-server/internal/domain"
)

type sqliteDialect struct{}

func init() {
	RegisterDialect("sqlite", &sqliteDialect{})
}

func (d *sqliteDialect) DriverName() string {
	return "sqlite"
}

func (d *sqliteDialect) Open(cfg domain.DatabaseConfig) (*sql.DB, error) {
	filePath := cfg.FilePath
	if filePath == "" {
		filePath = "monitor_logs.db"
	}

	db, err := sql.Open("sqlite", filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite ping failed: %w", err)
	}

	// Single open connection for SQLite ensures thread-safe sequential writes and avoids database locked errors
	db.SetMaxOpenConns(1)

	_, _ = db.Exec("PRAGMA foreign_keys = ON;")
	return db, nil
}

func (d *sqliteDialect) GetVersion(db *sql.DB) (string, error) {
	var version string
	if err := db.QueryRow("SELECT sqlite_version();").Scan(&version); err != nil {
		return "", err
	}
	return "SQLite " + version, nil
}

func (d *sqliteDialect) CheckSchemaExists(db *sql.DB, schema string) (bool, error) {
	return true, nil
}

func (d *sqliteDialect) InitSchema(db *sql.DB, schema string) error {
	return initSchema(db)
}
