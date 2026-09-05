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
// File: internal/tunnel/driver.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package tunnel

import "context"

// State represents the current lifecycle status of the remote tunnel.
type State string

const (
	StateOffline    State = "OFFLINE"
	StateConnecting State = "CONNECTING"
	StateOnline     State = "ONLINE"
	StateError      State = "ERROR"
)

// Status represents the public telemetry access view data.
type Status struct {
	State        State  `json:"state"`
	PublicURL    string `json:"public_url"`
	TelemetryURL string `json:"telemetry_url"`
	Domain       string `json:"domain"`
	BinaryFound  bool   `json:"binary_found"`
	ErrorMessage string `json:"error_message,omitempty"`
	StartedAt    string `json:"started_at,omitempty"`
}

// Config encapsulates parameters required to establish an external tunnel.
type Config struct {
	AuthToken string
	Domain    string
	LocalPort string
}

// Driver abstracts low-level reverse tunnel providers (e.g., Ngrok, Cloudflare, Localtunnel).
// Adheres to Open/Closed Principle (OCP) and Liskov Substitution Principle (LSP).
type Driver interface {
	// Name returns the identifier of the tunnel provider.
	Name() string

	// IsAvailable checks whether the underlying provider binary or CLI is installed on the host OS.
	IsAvailable() bool

	// Start launches the tunnel connection for the given configuration.
	Start(ctx context.Context, cfg Config) error

	// Stop terminates the underlying tunnel connection.
	Stop() error

	// GetPublicURL inspects the live tunnel and returns the public HTTPS URL.
	GetPublicURL(ctx context.Context) (string, error)

	// Wait blocks until the driver process exits, returning the exit error if unexpected.
	Wait() error
}

// Service defines the high-level contract consumed by HTTP Handlers.
// Adheres to Interface Segregation Principle (ISP) and Dependency Inversion Principle (DIP).
type Service interface {
	Start(authToken, domain string) error
	Stop() error
	GetStatus() Status
	IsBinaryAvailable() bool
}
