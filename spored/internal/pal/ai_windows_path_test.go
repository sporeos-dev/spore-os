// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build windows

package pal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWindowsExecutablePathComparisonNormalizesSeparatorsAndCase(t *testing.T) {
	expected := "C:/ProgramData/spore-os/store/spore/spore.exe"
	actual := `c:\programdata\spore-os\store\spore\spore.exe`

	if !SameExecutablePath(expected, actual) {
		t.Fatalf("expected path %q did not match actual path %q", expected, actual)
	}
}

func TestWindowsEndpointUsesProgramDataSporeSocket(t *testing.T) {
	expected := filepath.Join(os.Getenv("ProgramData"), "spore-os", "spore.sock")
	if FileSocket() != expected {
		t.Fatalf("socket path = %q, want %q", FileSocket(), expected)
	}
}
