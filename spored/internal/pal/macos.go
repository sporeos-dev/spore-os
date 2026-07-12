//go:build darwin

package pal

// #cgo LDFLAGS: -framework CoreFoundation
// #include <libproc.h>
import "C"

import (
	"errors"
	"fmt"
	"net"
	"syscall"
	"unsafe"
)

type macos struct {}

func newImpl() impl { return newMacos() }

func newMacos() *macos {
	return &macos {}
}

func (m *macos) commandOpenFileManager() string { return "open" }
func (m *macos) daemonUsername() string { return "_spore" }
func (m *macos) directoryLogging() string { return "/Library/Logs/spore-os" }
func (m *macos) directoryRoot() string { return "/Library/Application Support/spore-os" }

const localPeerPID = 2
func (m *macos) processPath(conn net.Conn) (string, error) { 
	
	//
	//
	// get PID
	//

	uc, ok := conn.(*net.UnixConn)
	if !ok {
		return "", errors.New("connection casting failure")
	}
	raw, err := uc.SyscallConn()
	if err != nil {
		return "", err
	}
	var pid int
	_ = raw.Control(func(fd uintptr) {
		pid, _ = syscall.GetsockoptInt(int(fd), syscall.AF_UNIX, localPeerPID)
	})
	
	//
	//
	// get path
	//

	buf := make([]byte, C.PROC_PIDPATHINFO_MAXSIZE)
	n := C.proc_pidpath(C.int(pid), unsafe.Pointer(&buf[0]), C.uint32_t(len(buf)))
	if n <= 0 {
		return "", fmt.Errorf("proc_pidpath failed for pid %d", pid)
	}
	return string(buf[:n]), nil
}
