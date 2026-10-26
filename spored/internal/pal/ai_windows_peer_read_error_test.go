// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build windows

package pal

import (
	"fmt"
	"testing"

	winapi "golang.org/x/sys/windows"
)

func TestWindowsTerminalPeerReadErrors(t *testing.T) {
	platform := newWindows()

	for _, err := range []error{
		winapi.WSAECONNABORTED,
		winapi.WSAECONNRESET,
		fmt.Errorf("wrapped: %w", winapi.ERROR_OPERATION_ABORTED),
	} {
		if !platform.isTerminalPeerReadError(err) {
			t.Fatalf("terminal peer read error %v was not recognized", err)
		}
	}
}
