// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// DataRoot returns the system-wide data root directory for spore-os.
//
// The installer creates this directory and owns its contents. spored reads
// from it at startup. SPORE_DATA_DIR overrides the root — used by tests and
// local dev runs to avoid touching system paths.
//
//	Linux:   /var/lib/spore-os
//	macOS:   /Library/Application Support/spore-os
//	Windows: %LOCALAPPDATA%\spore-os
func DataRoot() (string, error) {
	if override := os.Getenv("SPORE_DATA_DIR"); override != "" {
		return override, nil
	}
	switch runtime.GOOS {
	case "linux":
		return "/var/lib/spore-os", nil
	case "darwin":
		return "/Library/Application Support/spore-os", nil
	case "windows":
		localappdata := os.Getenv("LOCALAPPDATA")
		if localappdata == "" {
			return "", fmt.Errorf("registry: LOCALAPPDATA environment variable not set")
		}
		return filepath.Join(localappdata, "spore-os"), nil
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// RegistryPath returns the absolute path to nodes.registry.yaml at the data root.
func RegistryPath() (string, error) {
	root, err := DataRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "nodes.registry.yaml"), nil
}

// EnsureRegistryDir creates the directory containing regPath if it does not
// already exist. Called before any write to nodes.registry.yaml so the path
// is valid even on a first install via SPORE.node.install.
func EnsureRegistryDir(regPath string) error {
	return os.MkdirAll(filepath.Dir(regPath), 0700)
}
