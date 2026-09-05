// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems

package storage

import (
	"testing"
)

func TestAdaptQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		driver   string
		expected string
	}{
		{
			name:     "sqlite retains question marks",
			query:    "SELECT * FROM devices WHERE id = ? AND name = ?",
			driver:   "sqlite",
			expected: "SELECT * FROM devices WHERE id = ? AND name = ?",
		},
		{
			name:     "postgres converts placeholders",
			query:    "SELECT * FROM devices WHERE id = ? AND name = ?",
			driver:   "postgres",
			expected: "SELECT * FROM devices WHERE id = $1 AND name = $2",
		},
		{
			name:     "postgres ignores question marks inside string literals",
			query:    "SELECT * FROM devices WHERE status = 'is_active?' AND id = ?",
			driver:   "postgres",
			expected: "SELECT * FROM devices WHERE status = 'is_active?' AND id = $1",
		},
		{
			name:     "postgres adapts INSERT OR IGNORE",
			query:    "INSERT OR IGNORE INTO settings (id) VALUES (1);",
			driver:   "postgres",
			expected: "INSERT INTO settings (id) VALUES (1) ON CONFLICT DO NOTHING;",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := AdaptQuery(tc.query, tc.driver)
			if got != tc.expected {
				t.Errorf("AdaptQuery(%q, %q) = %q; want %q", tc.query, tc.driver, got, tc.expected)
			}
		})
	}
}
