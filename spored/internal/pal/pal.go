// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package pal

import (
	"net"
	"os"
	"path/filepath"
	"spored/internal/utilities/file"

	"github.com/kardianos/service"
)

type pal struct {
	impl impl
}

var instance *pal

func init() {
	instance = &pal{
		impl: newImpl(),
	}
}

// Platform behaviour
func CommandOpenFileManager() string { return instance.impl.commandOpenFileManager() }
func DaemonUsername() string         { return instance.impl.daemonUsername() }

// Directories
func DirectoryRoot() string      		{ return instance.impl.directoryRoot() }
func DirectoryStore() string     		{ return filepath.Join(instance.impl.directoryRoot(), "store") }
func DirectoryLogging() string   		{ return instance.impl.directoryLogging() }
func DirectoryData() string  { return filepath.Join(instance.impl.directoryRoot(), "data") }

// Files
func FileRegistry() string    { return filepath.Join(instance.impl.directoryRoot(), "nodes.registry.yaml") }
func FileSocket() string      { return filepath.Join(instance.impl.directoryRoot(), "spore.sock") }
func FileLog() string         { return filepath.Join(instance.impl.directoryLogging(), "dev.sporeos.spored.log") }
func FilePermissions() string { return filepath.Join(DirectoryData(), "permissions.yaml") }
func FileSporeManifest() string {
	if service.Interactive() {
		exe, err := os.Executable()
		if err != nil {
			return "spored.manifest.spore.yaml"
		}
		fp := filepath.Join(filepath.Dir(exe), "spored.manifest.spore.yaml")
		if !file.IsReadable(fp) {
			fp = filepath.Join(instance.impl.directoryRoot(), "spored.manifest.spore.yaml")
		}
		return fp
	} else {
		return filepath.Join(instance.impl.directoryRoot(), "spored.manifest.spore.yaml")
	}
}

// PeerPID
func ProcessPath(conn net.Conn) (string, error) { return instance.impl.processPath(conn) }
func ProcessID(conn net.Conn) (int, error) { return instance.impl.processId(conn) }