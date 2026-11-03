// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux

package pal

import (
	"errors"
	"fmt"
	"net"
	"os"

	"golang.org/x/sys/unix"
)

type linux struct{}

func newImpl() impl { return newLinux() }

func newLinux() *linux {
	return &linux{}
}

func (l *linux) commandOpenFileManager() string { return "xdg-open" }
func (l *linux) daemonUsername() string         { return "spore" }
func (l *linux) directoryLogging() string       { return "/var/log/spore-os" }
func (l *linux) directoryRoot() string          { return "/var/lib/spore-os" }

func (l *linux) listenLocal() (net.Listener, error) {
	listener, err := net.Listen("unix", FileSocket())
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(FileSocket(), 0777); err != nil {
		listener.Close()
		return nil, err
	}
	return listener, nil
}

func (l *linux) removeLocalEndpoint() error {
	err := os.Remove(FileSocket())
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (l *linux) isTerminalPeerReadError(error) bool { return false }

func (l *linux) processId(conn net.Conn) (int, error) {
	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		return -1, errors.New("connection casting failure")
	}

	rawConn, err := unixConn.SyscallConn()
	if err != nil {
		return -1, err
	}

	var (
		credentials *unix.Ucred
		socketErr   error
	)
	err = rawConn.Control(func(fd uintptr) {
		credentials, socketErr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	})
	if err != nil {
		return -1, err
	}
	if socketErr != nil {
		return -1, socketErr
	}
	if credentials == nil {
		return -1, errors.New("peer credentials unavailable")
	}

	return int(credentials.Pid), nil
}

func (l *linux) processPath(conn net.Conn) (string, error) {
	pid, err := l.processId(conn)
	if err != nil {
		return "", err
	}

	return os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
}
