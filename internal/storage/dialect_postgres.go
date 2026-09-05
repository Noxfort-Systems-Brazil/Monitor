// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: internal/storage/dialect_postgres.go
// Author: Gabriel Moraes
// Date: 2026-09-05

package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq" // Pure Go PostgreSQL Driver
	"noxfort-monitor-server/internal/domain"
)

type postgresDialect struct{}

func init() {
	RegisterDialect("postgres", &postgresDialect{})
}

func (d *postgresDialect) DriverName() string {
	return "postgres"
}

// BuildPostgresDSN formats the connection string for lib/pq.
func BuildPostgresDSN(cfg domain.DatabaseConfig) string {
	host := cfg.Host
	if host == "" {
		host = "localhost"
	}
	port := cfg.Port
	if port == 0 {
		port = 5432
	}
	user := cfg.User
	if user == "" {
		user = "postgres"
	}
	dbname := cfg.DBName
	if dbname == "" {
		dbname = "banco_de_dados_noxfort"
	}
	sslmode := cfg.SSLMode
	if sslmode == "" {
		sslmode = "disable"
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s connect_timeout=5",
		host, port, user, dbname, sslmode,
	)
	if cfg.Password != "" {
		dsn += fmt.Sprintf(" password=%s", cfg.Password)
	}

	schema := cfg.Schema
	if schema == "" {
		schema = "schema_monitor"
	}
	sanitizedSchema := strings.ReplaceAll(schema, "\"", "")
	dsn += fmt.Sprintf(" search_path=%s,public", sanitizedSchema)

	return dsn
}

func (d *postgresDialect) Open(cfg domain.DatabaseConfig) (*sql.DB, error) {
	dsn := BuildPostgresDSN(cfg)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	// Production-grade PostgreSQL Connection Pool Tuning
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(15 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("postgres ping failed: %w", err)
	}

	return db, nil
}

func (d *postgresDialect) GetVersion(db *sql.DB) (string, error) {
	var version string
	if err := db.QueryRow("SELECT version();").Scan(&version); err != nil {
		return "", err
	}
	return version, nil
}

func (d *postgresDialect) CheckSchemaExists(db *sql.DB, schema string) (bool, error) {
	return CheckPostgresSchemaExists(db, schema)
}

func (d *postgresDialect) InitSchema(db *sql.DB, schema string) error {
	return InitPostgresSchema(db, schema)
}
