// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux

package hub

import (
	"os"
	"path/filepath"
)

// socketPath returns the path for the Unix domain socket.
// /run/spore/ is the standard Linux location for system daemon sockets,
// equivalent to /var/run/spore/ (they are the same directory on modern systems).
// The directory is created by the installer owned by _spore; MkdirAll is a
// safety net only.
func socketPath() string {
	if override := os.Getenv("SPORE_DATA_DIR"); override != "" {
		dir := filepath.Join(override, "run")
		os.MkdirAll(dir, 0755)
		return filepath.Join(dir, "spore.sock")
	}
	const dir = "/run/spore"
	os.MkdirAll(dir, 0755)
	return dir + "/spore.sock"
}
