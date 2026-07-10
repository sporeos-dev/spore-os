package pal

type macos struct {}

func newMacos() *macos {
	return &macos {}
}

func (m *macos) commandOpenFileManager() string { return "open" }
func (m *macos) daemonUsername() string { return "_spore" }
func (m *macos) directoryLogging() string { return "/Library/Logs/spore-os" }
func (m *macos) directoryRoot() string { return "/Library/Application Support/spore-os" }