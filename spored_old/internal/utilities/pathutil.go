// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package utilities

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ResolveAppPath resolves the app: field of a manifest to an absolute path.
// base is the directory of the source manifest file.
//
// Forms handled:
//   - ""    — absent / not declared; returned unchanged.
//   - "n/a" — explicit sentinel for system nodes with no binary; returned unchanged.
//   - "~/…" — tilde-prefixed; ~ is expanded to the current user's home directory.
//   - "/…"  — already absolute; returned as-is (except on Windows where /usr/local/bin is mapped).
//   - other — relative path; joined with base.
func ResolveAppPath(app, base string) (string, error) {
	if app == "" || app == "n/a" {
		return app, nil
	}
	if len(app) >= 2 && app[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve app path: %w", err)
		}
		return filepath.Join(home, app[2:]), nil
	}

	// On Windows, dynamically translate POSIX system path /usr/local/bin/ to Windows local/programdata bin directory
	if runtime.GOOS == "windows" {
		cleaned := filepath.ToSlash(app)
		if strings.HasPrefix(cleaned, "/usr/local/bin/") {
			binaryName := strings.TrimPrefix(cleaned, "/usr/local/bin/")
			binBase := strings.ToLower(strings.TrimSuffix(binaryName, ".exe"))
			if binBase == "spore" || binBase == "spore-shell" || binBase == "spore-witness" || binBase == "spore-log" || binBase == "spore-dialog" {
				if !strings.HasSuffix(strings.ToLower(binaryName), ".exe") {
					binaryName += ".exe"
				}
				var winBinDir string
				if override := os.Getenv("SPORE_DATA_DIR"); override != "" {
					winBinDir = filepath.Join(override, "bin")
				} else {
					winBinDir = `C:\ProgramData\spore-os\bin`
				}
				return filepath.Join(winBinDir, binaryName), nil
			}
		}
	}

	if filepath.IsAbs(app) || (len(app) > 0 && (app[0] == '/' || app[0] == '\\')) {
		return app, nil
	}
	return filepath.Join(base, app), nil
}
