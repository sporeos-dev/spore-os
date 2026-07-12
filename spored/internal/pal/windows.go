//go:build windows

package pal

import (
	"errors"
	"net"
	"os"
)

type windows struct{}

func newImpl() impl { return newWindows() }

func newWindows() *windows {
	return &windows {}
}

func (w *windows) commandOpenFileManager() string { return "explorer" }
func (w *windows) daemonUsername() string { return "" }
func (w *windows) directoryLogging() string { return os.Getenv("LOCALAPPDATA") + `\spore-os\logs` }
func (w *windows) directoryRoot() string { return os.Getenv("LOCALAPPDATA") + `\spore-os` }

func (w *windows) processPath(conn net.Conn) (string, error) { return "", errors.New("not impl.") }