package pal

type impl interface {
	commandOpenFileManager() string
	daemonUsername() string
	directoryLogging() string
	directoryRoot() string
}