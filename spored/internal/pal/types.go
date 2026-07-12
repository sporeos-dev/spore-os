package pal

import (
	"net"
)

type impl interface {
	commandOpenFileManager() string
	daemonUsername() string
	directoryLogging() string
	directoryRoot() string
	processPath(net.Conn) (string, error)
}