// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux

package pal

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestLinuxPeerProcessIdentity(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "spored.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	connected := make(chan struct{})
	go func() {
		conn, dialErr := net.Dial("unix", socketPath)
		if dialErr == nil {
			close(connected)
			defer conn.Close()
		}
	}()

	conn, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	<-connected

	impl := newLinux()
	pid, err := impl.processId(conn)
	if err != nil {
		t.Fatal(err)
	}
	if pid != os.Getpid() {
		t.Fatalf("peer pid = %d, want %d", pid, os.Getpid())
	}

	peerPath, err := impl.processPath(conn)
	if err != nil {
		t.Fatal(err)
	}
	selfPath, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	peerInfo, err := os.Stat(peerPath)
	if err != nil {
		t.Fatal(err)
	}
	selfInfo, err := os.Stat(selfPath)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(peerInfo, selfInfo) {
		t.Fatalf("peer executable %q does not match test executable %q", peerPath, selfPath)
	}
}
