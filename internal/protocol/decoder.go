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
// File: internal/protocol/decoder.go
// Author: Gabriel Moraes
// Date: 2026-01-19

package protocol

import (
	"encoding/json"
	"fmt"

	"noxfort-monitor-server/internal/domain"
)

// DecodePayload transforms the raw MQTT JSON payload into a structured Domain Event.
// It expects a valid JSON string adhering to the IncomingEvent schema.
//
// Input: Raw bytes (e.g., `{"origin":"synapse", "level":"CRITICAL", ...}`)
// Output: Pointer to IncomingEvent struct or error if invalid.
func DecodePayload(payload []byte) (*domain.IncomingEvent, error) {
	// 1. Basic empty check
	// Prevents processing nil or zero-length packets immediately.
	if len(payload) == 0 {
		return nil, fmt.Errorf("payload is empty")
	}

	// 2. Unmarshal JSON directly into the Domain Entity
	// Go's standard library automatically handles ISO 8601 time strings
	// mapping them correctly to the time.Time field in the struct.
	var event domain.IncomingEvent
	err := json.Unmarshal(payload, &event)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON payload: %w", err)
	}

	// 3. Structural Validation
	// We ensure strictly required fields are present to avoid logic errors downstream.
	// Even though it is JSON, we enforce the protocol contract here.
	if event.Origin == "" {
		return nil, fmt.Errorf("missing required field: 'origin'")
	}
	if event.Level == "" {
		return nil, fmt.Errorf("missing required field: 'level'")
	}
	if event.Message == "" {
		return nil, fmt.Errorf("missing required field: 'message'")
	}

	return &event, nil
}
