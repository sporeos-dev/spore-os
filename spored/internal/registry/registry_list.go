// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// registryPath returns the absolute path to the manifest registry file.
//
// CURRENT IMPLEMENTATION: user-level daemon.
// This means spore-os is started by the user (e.g. from a terminal, login item,
// or user-space autostart). The process runs as that user, so it reads/writes
// inside the user's own config directory. No elevated privileges needed.
//
// The returned path will be something like:
//   Linux:   ~/.config/spore-os/manifest_registry
//   macOS:   ~/Library/Application Support/spore-os/manifest_registry
//   Windows: C:\Users\you\AppData\Roaming\spore-os\manifest_registry
//
// -----------------------------------------------------------------------
// FUTURE: system-level daemon notes
// -----------------------------------------------------------------------
// A system-level daemon is started by the OS itself at boot, before any user
// logs in (launchd on macOS, systemd on Linux, Windows Service on Windows).
// It runs as a privileged or dedicated service account, not as a regular user.
// os.UserConfigDir() would return an error or point to the wrong place.
//
// Instead, use a fixed system-wide path per platform. The simplest approach
// is a build-tag-free runtime switch:
//
//   Linux:   /var/lib/spore-os/manifest_registry
//             (owned by a dedicated 'sporeos' system user, mode 0600)
//   macOS:   /Library/Application Support/spore-os/manifest_registry
//             (system /Library, not ~/Library — requires root to create)
//   Windows: C:\ProgramData\spore-os\manifest_registry
//             (ProgramData is the system-wide equivalent of AppData\Roaming)
//
// You would replace the body of this function with:
//
//   switch runtime.GOOS {
//   case "linux":
//       return "/var/lib/spore-os/manifest_registry", nil
//   case "darwin":
//       return "/Library/Application Support/spore-os/manifest_registry", nil
//   case "windows":
//       return `C:\ProgramData\spore-os\manifest_registry`, nil
//   default:
//       return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
//   }
//
// The directory would be created by the installer (with the right ownership
// and permissions), not by this code at runtime.
// -----------------------------------------------------------------------
func RegistryPath() (string, error) {
	// SPORE_DATA_DIR overrides the data root — used by tests and local dev
	// to avoid touching system paths. Not set by launchd in production.
	if override := os.Getenv("SPORE_DATA_DIR"); override != "" {
		return filepath.Join(override, "data", "manifest_registry"), nil
	}
	switch runtime.GOOS {
	case "linux":
		return "/var/lib/spore-os/data/manifest_registry", nil
	case "darwin":
		return "/Library/Application Support/spore-os/data/manifest_registry", nil
	case "windows":
		return `C:\ProgramData\spore-os\data\manifest_registry`, nil
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// ManifestsDir returns the directory where installed node manifests are stored.
// On Add, the daemon copies the manifest here so it can always read it
// regardless of where the original file lives (e.g. a user's home directory
// that _spore has no access to).
func ManifestsDir() (string, error) {
	if override := os.Getenv("SPORE_DATA_DIR"); override != "" {
		return filepath.Join(override, "manifests"), nil
	}
	switch runtime.GOOS {
	case "linux":
		return "/var/lib/spore-os/manifests", nil
	case "darwin":
		return "/Library/Application Support/spore-os/manifests", nil
	case "windows":
		return `C:\ProgramData\spore-os\manifests`, nil
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// ensureRegistryDir creates the directory containing the registry file if it
// does not already exist. Safe to call every startup — MkdirAll is a no-op
// when the directory is already present (like `mkdir -p`).
//
// 0700 means only the owner can read, write, or enter this directory.
// Group and others have no access at all.
func EnsureRegistryDir(regPath string) error {
	dir := filepath.Dir(regPath) // strips the filename, keeps the directory portion
	return os.MkdirAll(dir, 0700)
}
