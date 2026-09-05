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
// File: internal/tunnel/ngrok_driver_test.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package tunnel

import (
	"context"
	"testing"
)

func TestNgrokDriver_Name(t *testing.T) {
	d := NewNgrokDriver()
	if d.Name() != "ngrok" {
		t.Errorf("Expected driver name 'ngrok', got '%s'", d.Name())
	}
}

func TestNgrokDriver_Validation(t *testing.T) {
	d := NewNgrokDriver()

	// Starting without token must return validation error
	err := d.Start(context.Background(), Config{
		AuthToken: "",
		Domain:    "test.domain",
		LocalPort: "8080",
	})

	if err == nil && d.IsAvailable() {
		t.Errorf("Expected error when starting ngrok without authtoken, got nil")
	}
}
