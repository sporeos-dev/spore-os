//go:build !darwin

package connection

import "net"

// peerPID is not implemented on this platform.
func peerPID(conn net.Conn) int { return 0 }
