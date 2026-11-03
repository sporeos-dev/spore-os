// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package pal

import (
	"net"
)

type impl interface {
	commandOpenFileManager() string
	daemonUsername() string
	directoryLogging() string
	directoryRoot() string
	listenLocal() (net.Listener, error)
	processId(net.Conn) (int, error)
	processPath(net.Conn) (string, error)
	removeLocalEndpoint() error
	isTerminalPeerReadError(error) bool
}
