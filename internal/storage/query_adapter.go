// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: internal/storage/query_adapter.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package storage

import (
	"fmt"
	"strings"
)

// AdaptQuery rewrites SQL query placeholders (? to $1, $2...) when the active dialect is PostgreSQL.
// It also adapts standard syntax differences (e.g., INSERT OR IGNORE -> ON CONFLICT DO NOTHING).
func AdaptQuery(query string, driver string) string {
	if driver != "postgres" {
		return query
	}

	var sb strings.Builder
	paramIndex := 1
	inSingleQuote := false

	// Replace INSERT OR IGNORE with ON CONFLICT DO NOTHING if present
	adapted := query
	if strings.Contains(strings.ToUpper(adapted), "INSERT OR IGNORE INTO") {
		// Replace "INSERT OR IGNORE INTO table (col) VALUES (val)" with "INSERT INTO table (col) VALUES (val) ON CONFLICT DO NOTHING"
		adapted = strings.ReplaceAll(adapted, "INSERT OR IGNORE INTO", "INSERT INTO")
		adapted = strings.ReplaceAll(adapted, "insert or ignore into", "insert into")
		trimmed := strings.TrimRight(strings.TrimSpace(adapted), ";")
		adapted = trimmed + " ON CONFLICT DO NOTHING;"
	}

	for i := 0; i < len(adapted); i++ {
		char := adapted[i]
		if char == '\'' {
			inSingleQuote = !inSingleQuote
			sb.WriteByte(char)
			continue
		}

		if char == '?' && !inSingleQuote {
			sb.WriteString(fmt.Sprintf("$%d", paramIndex))
			paramIndex++
		} else {
			sb.WriteByte(char)
		}
	}

	return sb.String()
}
