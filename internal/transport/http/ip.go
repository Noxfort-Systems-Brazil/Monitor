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
// File: internal/transport/http/ip.go
// Author: Gabriel Moraes
// Date: 2026-01-20

package http

import (
	"net"
)

// GetLocalIP returns the primary outbound local IPv4 address of the host machine.
func GetLocalIP() string {
	// Try establishing a dummy UDP connection to determine preferred outbound interface IP
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		localAddr := conn.LocalAddr().(*net.UDPAddr)
		if ip := localAddr.IP.To4(); ip != nil {
			return ip.String()
		}
	}

	// Fallback: enumerate non-loopback network interfaces
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, address := range addrs {
			if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ip := ipnet.IP.To4(); ip != nil {
					return ip.String()
				}
			}
		}
	}

	return "127.0.0.1"
}
