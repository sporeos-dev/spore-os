// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build !windows

package pal

func SameExecutablePath(expected, actual string) bool {
	return expected == actual
}
