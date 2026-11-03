// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build windows

package pal

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"unsafe"

	winapi "golang.org/x/sys/windows"
)

type windows struct{}

const sioAFUnixGetPeerPID = 0x58000100

const socketSecurityDescriptor = "D:P(A;;GA;;;SY)(A;;GA;;;BA)(A;;GA;;;LS)(A;;GRGW;;;AU)"

func newImpl() impl { return newWindows() }

func newWindows() *windows {
	return &windows{}
}

func (w *windows) commandOpenFileManager() string { return "explorer" }
func (w *windows) daemonUsername() string         { return `NT AUTHORITY\LocalService` }
func (w *windows) directoryLogging() string       { return filepath.Join(w.directoryRoot(), "logs") }
func (w *windows) directoryRoot() string          { return filepath.Join(os.Getenv("ProgramData"), "spore-os") }

func (w *windows) listenLocal() (net.Listener, error) {
	listener, err := net.Listen("unix", FileSocket())
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(FileSocket(), 0777); err != nil {
		listener.Close()
		return nil, err
	}
	if err := setSocketAccess(FileSocket()); err != nil {
		listener.Close()
		return nil, err
	}
	return listener, nil
}

func setSocketAccess(path string) error {
	descriptor, err := winapi.SecurityDescriptorFromString(socketSecurityDescriptor)
	if err != nil {
		return err
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		return err
	}
	return winapi.SetNamedSecurityInfo(
		path,
		winapi.SE_FILE_OBJECT,
		winapi.DACL_SECURITY_INFORMATION|winapi.PROTECTED_DACL_SECURITY_INFORMATION,
		nil,
		nil,
		dacl,
		nil,
	)
}

func (w *windows) removeLocalEndpoint() error {
	err := os.Remove(FileSocket())
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (w *windows) isTerminalPeerReadError(err error) bool {
	return errors.Is(err, winapi.WSAECONNABORTED) ||
		errors.Is(err, winapi.WSAECONNRESET) ||
		errors.Is(err, winapi.ERROR_OPERATION_ABORTED)
}

func (w *windows) processId(conn net.Conn) (int, error) {
	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		return -1, errors.New("connection casting failure")
	}

	rawConn, err := unixConn.SyscallConn()
	if err != nil {
		return -1, err
	}

	var (
		pid       uint32
		bytesRead uint32
		socketErr error
	)
	err = rawConn.Control(func(fd uintptr) {
		socketErr = winapi.WSAIoctl(
			winapi.Handle(fd),
			sioAFUnixGetPeerPID,
			nil,
			0,
			(*byte)(unsafe.Pointer(&pid)),
			uint32(unsafe.Sizeof(pid)),
			&bytesRead,
			nil,
			0)
	})
	if err != nil {
		return -1, err
	}
	if socketErr != nil {
		return -1, socketErr
	}
	if pid == 0 {
		return -1, fmt.Errorf("AF_UNIX peer PID query returned %d bytes and PID %d", bytesRead, pid)
	}

	return int(pid), nil
}

func (w *windows) processPath(conn net.Conn) (string, error) {
	pid, err := w.processId(conn)
	if err != nil {
		return "", err
	}

	process, err := winapi.OpenProcess(winapi.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		if errors.Is(err, winapi.ERROR_ACCESS_DENIED) {
			return "", fmt.Errorf("%w: %w", ErrPeerProcessPathAccessDenied, err)
		}
		return "", err
	}
	defer winapi.CloseHandle(process)

	path := make([]uint16, winapi.MAX_LONG_PATH)
	pathLength := uint32(len(path))
	if err := winapi.QueryFullProcessImageName(process, 0, &path[0], &pathLength); err != nil {
		return "", err
	}
	return winapi.UTF16ToString(path[:pathLength]), nil
}
