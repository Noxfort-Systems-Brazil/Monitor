// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: internal/storage/postgres_admin.go
// Author: Gabriel Moraes
// Date: 2026-09-05

package storage

import (
	"fmt"
	"log"
	"strings"

	"noxfort-monitor-server/internal/domain"
)

// ProvisionPostgresUser connects using administrative credentials to create a new database user and grant permissions.
func ProvisionPostgresUser(host string, port int, dbname, adminUser, adminPassword, newUser, newPassword, sslmode string) error {
	if host == "" {
		host = "localhost"
	}
	if port == 0 {
		port = 5432
	}
	if sslmode == "" {
		sslmode = "disable"
	}
	if adminUser == "" {
		adminUser = "postgres"
	}

	// 1. Connect to admin DB ("postgres" or dbname) as adminUser
	adminCfg := domain.DatabaseConfig{
		Type:     "postgres",
		Host:     host,
		Port:     port,
		DBName:   "postgres",
		User:     adminUser,
		Password: adminPassword,
		SSLMode:  sslmode,
	}

	adminDB, _, err := OpenConnection(adminCfg)
	if err != nil {
		// Try connecting directly to target dbname if "postgres" database is not default
		adminCfg.DBName = dbname
		adminDB, _, err = OpenConnection(adminCfg)
		if err != nil {
			return fmt.Errorf("falha ao conectar como administrador '%s': %w", adminUser, err)
		}
	}
	defer adminDB.Close()

	if strings.ContainsAny(newUser, " \t\n\r;\"'\\") {
		return fmt.Errorf("nome de usuário inválido: '%s'", newUser)
	}

	// 2. Create role if not exists or update password
	queryUser := fmt.Sprintf(`
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = %s) THEN
        EXECUTE 'CREATE ROLE ' || quote_ident(%s) || ' WITH LOGIN PASSWORD ' || quote_literal(%s);
    ELSE
        EXECUTE 'ALTER ROLE ' || quote_ident(%s) || ' WITH LOGIN PASSWORD ' || quote_literal(%s);
    END IF;
END
$$;`, quoteLiteral(newUser), quoteLiteral(newUser), quoteLiteral(newPassword), quoteLiteral(newUser), quoteLiteral(newPassword))

	if _, err := adminDB.Exec(queryUser); err != nil {
		return fmt.Errorf("falha ao criar/atualizar usuário '%s': %w", newUser, err)
	}

	// 3. Ensure target database exists and grant permissions
	if dbname != "" && dbname != "postgres" {
		if strings.ContainsAny(dbname, " \t\n\r;\"'\\") {
			return fmt.Errorf("nome de banco de dados inválido: '%s'", dbname)
		}
		var exists bool
		_ = adminDB.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1);", dbname).Scan(&exists)
		if !exists {
			if _, err := adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s OWNER %s;", quoteIdent(dbname), quoteIdent(newUser))); err != nil {
				log.Printf("[PROVISION] Warning creating database %s: %v", dbname, err)
			}
		}

		grantDBQuery := fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s;", quoteIdent(dbname), quoteIdent(newUser))
		if _, err := adminDB.Exec(grantDBQuery); err != nil {
			log.Printf("[PROVISION] Warning granting privileges on db %s: %v", dbname, err)
		}
	}

	log.Printf("[PROVISION] Usuário '%s' provisionado com sucesso no PostgreSQL.", newUser)
	return nil
}

func quoteLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
