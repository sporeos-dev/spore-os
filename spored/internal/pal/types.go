package pal

import (
	"net"
)

type impl interface {
	commandOpenFileManager() string
	daemonUsername() string
	directoryLogging() string
	directoryRoot() string
	processId(net.Conn) (int, error)
	processPath(net.Conn) (string, error)
}