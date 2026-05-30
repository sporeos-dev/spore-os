// Copyright 2026 mharr
// SPDX-License-Identifier: AGPL-3.0-only

package utilities

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolveAppPath resolves the app: field of a manifest to an absolute path.
// base is the directory of the source manifest file.
//
// Forms handled:
//   - ""    — absent / not declared; returned unchanged.
//   - "n/a" — explicit sentinel for system nodes with no binary; returned unchanged.
//   - "~/…" — tilde-prefixed; ~ is expanded to the current user's home directory.
//   - "/…"  — already absolute; returned as-is.
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
	if filepath.IsAbs(app) {
		return app, nil
	}
	return filepath.Join(base, app), nil
}
