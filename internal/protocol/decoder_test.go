// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.
//
// File: internal/protocol/decoder_test.go
// Author: Gabriel Moraes
// Date: 2026-01-19

package protocol

import (
	"testing"
	"time"

	"noxfort-monitor-server/internal/domain"
)

// TestDecodePayload_Valid verifies if a correctly formed JSON is parsed into the struct fields.
func TestDecodePayload_Valid(t *testing.T) {
	// 1. Prepare a Mock JSON Payload
	jsonPayload := []byte(`{
		"origin": "synapse",
		"level": "CRITICAL",
		"message": "Camera signal lost",
		"occurred_at": "2026-01-19T14:30:00Z"
	}`)

	// 2. Run the Function Under Test
	result, err := DecodePayload(jsonPayload)

	// 3. Assertions
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.Origin != "synapse" {
		t.Errorf("Expected Origin 'synapse', got '%s'", result.Origin)
	}

	if result.Level != domain.LevelCritical {
		t.Errorf("Expected Level 'CRITICAL', got '%s'", result.Level)
	}

	if result.Message != "Camera signal lost" {
		t.Errorf("Expected Message 'Camera signal lost', got '%s'", result.Message)
	}

	// Verify time parsing
	expectedTime, _ := time.Parse(time.RFC3339, "2026-01-19T14:30:00Z")
	if !result.OccurredAt.Equal(expectedTime) {
		t.Errorf("Expected Time %v, got %v", expectedTime, result.OccurredAt)
	}
}

// TestDecodePayload_InvalidJSON ensures the decoder rejects malformed strings.
func TestDecodePayload_InvalidJSON(t *testing.T) {
	malformedJSON := []byte(`{"origin": "synapse", "level": "INFO"`) // Missing closing brace

	_, err := DecodePayload(malformedJSON)

	if err == nil {
		t.Error("Expected error for malformed JSON, got nil")
	}
}

// TestDecodePayload_MissingFields ensures we strictly enforce required fields.
func TestDecodePayload_MissingFields(t *testing.T) {
	// Missing 'message'
	incompleteJSON := []byte(`{
		"origin": "synapse",
		"level": "INFO",
		"occurred_at": "2026-01-19T14:30:00Z"
	}`)

	_, err := DecodePayload(incompleteJSON)

	if err == nil {
		t.Error("Expected error for missing required fields, got nil")
	}

	expectedErrorFragment := "missing required field"
	if err != nil && err.Error() != "missing required field: 'message'" {
		t.Logf("Got error: %v (Expected to contain '%s')", err, expectedErrorFragment)
	}
}

// TestDecodePayload_EmptyPayload checks nil handling.
func TestDecodePayload_EmptyPayload(t *testing.T) {
	_, err := DecodePayload([]byte{})

	if err == nil {
		t.Error("Expected error for empty payload, got nil")
	}
}
