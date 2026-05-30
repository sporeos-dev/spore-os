// Copyright 2026 mharr
// SPDX-License-Identifier: AGPL-3.0-only

//go:build darwin

package connection

import (
	"net"
	"syscall"
)

// LOCAL_PEERPID is a macOS getsockopt option (level AF_UNIX) that returns
// the OS-assigned PID of the process on the other end of a Unix socket.
// This is a read-only kernel value — the peer cannot forge it.
const localPeerPID = 2

// peerPID extracts the peer process ID from a Unix socket connection.
// Returns 0 if the PID cannot be determined.
func peerPID(conn net.Conn) int {
	uc, ok := conn.(*net.UnixConn)
	if !ok {
		return 0
	}
	raw, err := uc.SyscallConn()
	if err != nil {
		return 0
	}
	var pid int
	_ = raw.Control(func(fd uintptr) {
		pid, _ = syscall.GetsockoptInt(int(fd), syscall.AF_UNIX, localPeerPID)
	})
	return pid
}
