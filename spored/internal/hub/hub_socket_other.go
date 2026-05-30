// Copyright 2026 Matt HarrisonHarrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build !linux

package hub

import (
	"os"
	"path/filepath"
	"runtime"
)

// socketPath returns the path for the Unix domain socket.
//
// System-level daemon: the socket lives in a directory created by the
// installer and owned by _spore. The daemon creates the socket file itself.
// The directory is 0755 so that client processes (running as regular users)
// can reach the socket. The socket is world-connectable — peer credentials
// are the actual access gate, not filesystem permissions.
//
//	macOS:   /Library/Application Support/spore-os/run/spore.sock
//	Windows: C:\ProgramData\spore-os\spore.sock
//
// Note: /var/run is cleared by macOS on every boot and the _spore user lacks
// permission to recreate it. /Library/Application Support/spore-os/ is owned
// by _spore, persists across reboots, and requires no extra setup plist.
func socketPath() string {
	switch runtime.GOOS {
	case "darwin":
		dir := "/Library/Application Support/spore-os/run"
		os.MkdirAll(dir, 0755)
		return filepath.Join(dir, "spore.sock")
	case "windows":
		dir := `C:\ProgramData\spore-os`
		os.MkdirAll(dir, 0755)
		return filepath.Join(dir, "spore.sock")
	default:
		return "/tmp/spore.sock"
	}
}
