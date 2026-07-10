package pal

import "os"

type windows struct{}

func newWindows() *windows {
	return &windows {}
}

func (w *windows) commandOpenFileManager() string { return "explorer" }
func (w *windows) daemonUsername() string { return "" }
func (w *windows) directoryLogging() string { return os.Getenv("LOCALAPPDATA") + `\spore-os\logs` }
func (w *windows) directoryRoot() string { return os.Getenv("LOCALAPPDATA") + `\spore-os` }
