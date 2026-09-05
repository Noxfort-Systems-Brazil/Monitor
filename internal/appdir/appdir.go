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
// File: internal/appdir/appdir.go
// Author: Gabriel Moraes
// Date: 2026-01-20
//
// Resolves the application's asset root directory.
// Priority: NOXFORT_HOME env var → directory of the running executable → working directory.

package appdir

import (
	"os"
	"path/filepath"
)

var root string

func init() {
	if env := os.Getenv("NOXFORT_HOME"); env != "" {
		root = env
		return
	}

	candidates := []string{}

	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		dir := filepath.Dir(exe)
		candidates = append(candidates, dir)
		candidates = append(candidates, filepath.Dir(dir)) // for bin/noxfort-monitor
	}

	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, cwd)
		curr := cwd
		for {
			parent := filepath.Dir(curr)
			if parent == curr {
				break
			}
			candidates = append(candidates, parent)
			curr = parent
		}
	}

	candidates = append(candidates, "/opt/noxfort-monitor", ".")

	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "web", "templates", "layout.html")); err == nil {
			root = c
			return
		}
	}

	root = "."
}

// SetRoot allows explicitly overriding the root directory (useful in tests).
func SetRoot(r string) {
	root = r
}

// Root returns the current resolved root directory.
func Root() string {
	return root
}

// Path joins the app root with the given path segments.
func Path(parts ...string) string {
	all := append([]string{root}, parts...)
	return filepath.Join(all...)
}

