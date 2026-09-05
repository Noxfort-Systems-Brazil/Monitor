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
// File: internal/tunnel/ngrok_driver.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package tunnel

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// CleanAuthToken strips command prefixes, whitespace, and quotes from user-provided authtokens.
func CleanAuthToken(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "ngrok config add-authtoken")
	raw = strings.TrimPrefix(raw, "ngrok authtoken")
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, `"'`)
	return strings.TrimSpace(raw)
}

// CleanDomain strips scheme prefixes, paths, and trailing slashes from domain names.
func CleanDomain(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "https://")
	raw = strings.TrimPrefix(raw, "http://")
	raw = strings.TrimRight(raw, "/")
	return strings.TrimSpace(raw)
}

// NgrokDriver handles execution and inspection of the Ngrok tunnel CLI.
// Implements the Driver interface (Single Responsibility: Ngrok provider mechanics).
type NgrokDriver struct {
	mu         sync.Mutex
	cmd        *exec.Cmd
	cancel     context.CancelFunc
	httpClient *http.Client
	stderrBuf  strings.Builder
}

// NewNgrokDriver creates a new NgrokDriver instance.
func NewNgrokDriver() *NgrokDriver {
	return &NgrokDriver{
		httpClient: &http.Client{Timeout: 1 * time.Second},
	}
}

// Name returns the provider identifier.
func (d *NgrokDriver) Name() string {
	return "ngrok"
}

// IsAvailable checks whether the ngrok binary is present in the host system's PATH.
func (d *NgrokDriver) IsAvailable() bool {
	_, err := exec.LookPath("ngrok")
	return err == nil
}

// Start configures the authtoken and launches the ngrok background process.
func (d *NgrokDriver) Start(ctx context.Context, cfg Config) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.IsAvailable() {
		return fmt.Errorf("ngrok binary not found in system PATH")
	}

	cleanToken := CleanAuthToken(cfg.AuthToken)
	if cleanToken == "" {
		return fmt.Errorf("ngrok authtoken is required")
	}

	// 1. Configure the authtoken via CLI
	cfgCmd := exec.Command("ngrok", "config", "add-authtoken", cleanToken)
	if out, err := cfgCmd.CombinedOutput(); err != nil {
		log.Printf("[TUNNEL-NGROK] Warning setting authtoken: %v (output: %s)", err, string(out))
	}

	// 2. Prepare command arguments
	cleanDomain := CleanDomain(cfg.Domain)
	args := []string{"http"}
	if cleanDomain != "" {
		args = append(args, "--url=https://"+cleanDomain)
	}
	port := cfg.LocalPort
	if port == "" {
		port = "8080"
	}
	args = append(args, port)

	procCtx, cancel := context.WithCancel(ctx)
	d.cancel = cancel
	cmd := exec.CommandContext(procCtx, "ngrok", args...)
	d.stderrBuf.Reset()
	cmd.Stderr = &d.stderrBuf
	d.cmd = cmd

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ngrok process: %w", err)
	}

	log.Printf("[TUNNEL-NGROK] Process started (PID: %d) on port %s (domain: %s)", cmd.Process.Pid, port, cleanDomain)
	return nil
}

// Stop cleanly kills the running ngrok subprocess.
func (d *NgrokDriver) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.cancel != nil {
		d.cancel()
		d.cancel = nil
	}

	if d.cmd != nil && d.cmd.Process != nil {
		log.Println("[TUNNEL-NGROK] Terminating ngrok subprocess...")
		err := d.cmd.Process.Kill()
		d.cmd = nil
		return err
	}

	return nil
}

// Wait blocks until the ngrok subprocess exits.
func (d *NgrokDriver) Wait() error {
	d.mu.Lock()
	cmd := d.cmd
	d.mu.Unlock()

	if cmd == nil {
		return nil
	}
	err := cmd.Wait()
	if err != nil {
		d.mu.Lock()
		errMsg := strings.TrimSpace(d.stderrBuf.String())
		d.mu.Unlock()
		if errMsg != "" {
			log.Printf("[TUNNEL-NGROK] Process output: %s", errMsg)
		}
	}
	return err
}

// GetPublicURL inspects the local ngrok client API at http://127.0.0.1:4040/api/tunnels.
func (d *NgrokDriver) GetPublicURL(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:4040/api/tunnels", nil)
	if err != nil {
		return "", err
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ngrok local API unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status from ngrok API: %d", resp.StatusCode)
	}

	var result struct {
		Tunnels []struct {
			PublicURL string `json:"public_url"`
			Proto     string `json:"proto"`
		} `json:"tunnels"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode ngrok tunnels response: %w", err)
	}

	for _, t := range result.Tunnels {
		if strings.HasPrefix(t.PublicURL, "https://") {
			return t.PublicURL, nil
		}
	}

	if len(result.Tunnels) > 0 && result.Tunnels[0].PublicURL != "" {
		return result.Tunnels[0].PublicURL, nil
	}

	return "", fmt.Errorf("no active tunnels found")
}
