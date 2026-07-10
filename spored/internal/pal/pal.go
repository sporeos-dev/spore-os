package pal

import (
	"path/filepath"
	"runtime"
)

type pal struct {
	impl impl
}

var instance *pal

func init() {
	var impl impl
	switch runtime.GOOS {
	case "darwin":
		impl = newMacos()
	case "linux":
		impl = newLinux()
	case "windows":
		impl = newWindows()
	default:
		panic("unsupported operating system: " + runtime.GOOS)
	}

	instance = &pal{
		impl: impl,
	}
}

// Platform behaviour
func CommandOpenFileManager() string { return instance.impl.commandOpenFileManager() }
func DaemonUsername() string         { return instance.impl.daemonUsername() }

// Directories
func DirectoryRoot() string    { return instance.impl.directoryRoot() }
func DirectoryData() string    { return filepath.Join(instance.impl.directoryRoot(), "data") }
func DirectoryStore() string   { return filepath.Join(instance.impl.directoryRoot(), "store") }
func DirectoryLogging() string { return instance.impl.directoryLogging() }

// Files
func FileRegistry() string { return filepath.Join(instance.impl.directoryRoot(), "nodes.registry.yaml") }
func FileSocket() string   { return filepath.Join(instance.impl.directoryRoot(), "spored.sock") }
func FileLog() string      { return filepath.Join(instance.impl.directoryLogging(), "dev.sporeos.spored.log") }