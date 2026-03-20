// File: internal/appdir/appdir.go
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
	exe, err := os.Executable()
	if err == nil {
		// resolve symlinks (e.g. /usr/local/bin → /opt/noxfort-monitor/)
		exe, _ = filepath.EvalSymlinks(exe)
		root = filepath.Dir(exe)
		return
	}
	root = "."
}

// Path joins the app root with the given path segments.
func Path(parts ...string) string {
	all := append([]string{root}, parts...)
	return filepath.Join(all...)
}
