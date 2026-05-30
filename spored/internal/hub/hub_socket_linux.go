// Copyright 2026 mharr
// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux

package hub

import (
	"os"
)

// socketPath returns the path for the Unix domain socket.
// /run/spore/ is the standard Linux location for system daemon sockets,
// equivalent to /var/run/spore/ (they are the same directory on modern systems).
// The directory is created by the installer owned by _spore; MkdirAll is a
// safety net only.
func socketPath() string {
	const dir = "/run/spore"
	os.MkdirAll(dir, 0755)
	return dir + "/spore.sock"
}
