// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build windows

package pal

import (
	"net"
	"path/filepath"
	"testing"
)

func TestSocketAccessAllowsAuthenticatedUsers(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "spored.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	if err := setSocketAccess(socketPath); err != nil {
		t.Fatal(err)
	}
}
