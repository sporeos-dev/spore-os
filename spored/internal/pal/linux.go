//go:build linux

package pal

import (
	"errors"
	"net"
)

type linux struct {}

func newImpl() impl { return newLinux() }

func newLinux() *linux {
	return &linux {}
}

func (l *linux) commandOpenFileManager() string { return "xdg-open" }
func (l *linux) daemonUsername() string { return "spore" }
func (l *linux) directoryLogging() string { return "/var/log/spore-os" }
func (l *linux) directoryRoot() string { return "/var/lib/spore-os" }

func (l *linux) processPath(conn net.Conn) (string, error) { return "", errors.New("not impl.") } 