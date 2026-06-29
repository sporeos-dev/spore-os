// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux

package hub

import (
	"os"
	"path/filepath"
	"spored/internal/registry"
)

// socketPath returns the path for the Unix domain socket.
// The socket lives at the data root: /var/lib/spore-os/spore.sock.
// The directory is created by the installer; MkdirAll is a safety net only.
func socketPath() string {
	if root, err := registry.DataRoot(); err == nil {
		os.MkdirAll(root, 0755)
		return filepath.Join(root, "spore.sock")
	}
	return "/tmp/spore.sock"
}
