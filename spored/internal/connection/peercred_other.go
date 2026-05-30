// Copyright 2026 mharr
// SPDX-License-Identifier: AGPL-3.0-only

//go:build !darwin

package connection

import "net"

// peerPID is not implemented on this platform.
func peerPID(conn net.Conn) int { return 0 }
