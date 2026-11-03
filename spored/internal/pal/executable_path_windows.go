// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build windows

package pal

import (
	"path/filepath"
	"strings"
)

func SameExecutablePath(expected, actual string) bool {
	return strings.EqualFold(filepath.Clean(expected), filepath.Clean(actual))
}
