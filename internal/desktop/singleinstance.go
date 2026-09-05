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
// File: internal/desktop/singleinstance.go
// Author: Gabriel Moraes
// Date: 2026-09-03

package desktop

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func getSocketPath() string {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir != "" {
		return filepath.Join(runtimeDir, "noxfort-monitor.sock")
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("noxfort-monitor-%d.sock", os.Getuid()))
}

// TryActivateExisting checks if another instance of Noxfort Monitor is already running.
// If an instance is active, it sends an activation command to bring the existing window to the front and returns true.
// If no instance is active, it returns false.
func TryActivateExisting() bool {
	sockPath := getSocketPath()
	conn, err := net.DialTimeout("unix", sockPath, 400*time.Millisecond)
	if err != nil {
		return false
	}
	defer conn.Close()

	_, _ = conn.Write([]byte("ACTIVATE\n"))
	return true
}

type singleInstanceServer struct {
	listener net.Listener
	sockPath string
	closeMu  sync.Mutex
	closed   bool
}

func (s *singleInstanceServer) Close() error {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	err := s.listener.Close()
	_ = os.Remove(s.sockPath)
	return err
}

// StartSingleInstanceServer starts listening for activation commands from subsequent invocations.
func StartSingleInstanceServer(onActivate func()) (io.Closer, error) {
	sockPath := getSocketPath()

	// Clean up stale socket file if not responding
	_ = os.Remove(sockPath)

	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on single instance socket: %w", err)
	}

	server := &singleInstanceServer{
		listener: listener,
		sockPath: sockPath,
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 64)
				n, _ := c.Read(buf)
				msg := strings.TrimSpace(string(buf[:n]))
				if msg == "ACTIVATE" || msg == "SHOW" {
					log.Println("[DESKTOP] Sinal de ativação recebido de outra instância. Restaurando janela...")
					if onActivate != nil {
						onActivate()
					}
				}
			}(conn)
		}
	}()

	return server, nil
}
