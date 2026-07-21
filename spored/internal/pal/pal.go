package pal

import (
	"net"
	"os"
	"path/filepath"

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
func DirectoryRoot() string      { return instance.impl.directoryRoot() }
func DirectoryData() string      { return filepath.Join(instance.impl.directoryRoot(), "data") }
func DirectoryStore() string     { return filepath.Join(instance.impl.directoryRoot(), "store") }
func DirectoryLogging() string   { return instance.impl.directoryLogging() }

// Files
func FileRegistry() string { return filepath.Join(instance.impl.directoryRoot(), "nodes.registry.yaml") }
func FileSocket() string   { return filepath.Join(instance.impl.directoryRoot(), "spore.sock") }
func FileLog() string      { return filepath.Join(instance.impl.directoryLogging(), "dev.sporeos.spored.log") }
func FileSporeManifest() string {
	if service.Interactive() {
		exe, err := os.Executable()
		if err == nil {
			return filepath.Join(filepath.Dir(exe), "spored.manifest.spore.yaml")
		}
		return "spored.manifest.spore.yaml"
	} else {
		return filepath.Join(instance.impl.directoryRoot(), "spored.manifest.spore.yaml")
	}
}

// PeerPID
func ProcessPath(conn net.Conn) (string, error) { return instance.impl.processPath(conn) }
func ProcessID(conn net.Conn) (int, error) { return instance.impl.processId(conn) }