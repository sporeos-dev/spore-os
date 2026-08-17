// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"spored/internal/logging"
	"spored/internal/pal"
	"syscall"

	"github.com/kardianos/service"
)

var svcConfig = &service.Config{
	Name:        "dev.sporeos.spored",
	DisplayName: "Spore OS Daemon",
	Description: "The Spore OS IPC hub daemon.",
	// UserName runs the daemon as the dedicated _spore system account.
	// This account is created by the installer and owns all spore data paths.
	// macOS: /Library/LaunchDaemons/ (requires sudo, runs at boot as _spore)
	// Linux: /etc/systemd/system/   (requires sudo, runs at boot as _spore)
	UserName: pal.DaemonUsername(),
	Option: service.KeyValue{
		// kardianos/service defaults system daemon logs to /var/log which
		// _spore cannot write to. Use a directory the installer creates
		// with _spore ownership.
		"LogDirectory": pal.DirectoryLogging(),
	},
}

func main() {

	//
	//
	// Handle subcommands before anything else.
	// service.Interactive() is true when run from a terminal (e.g. sudo spored install),
	// so subcommand handling must live here — outside the interactive/daemon branch —
	// otherwise commands like "install" are never reached from a shell.
	//
	if len(os.Args) > 1 {
		svc, err := service.New(&program{}, svcConfig)
		if err != nil {
			log.Fatalf("Failed to create service: %v", err)
		}

		if os.Args[1] == "home" {
			root := pal.DirectoryRoot()
			open := pal.CommandOpenFileManager()
			cmd := exec.Command(open, root)
			if cmd != nil {
				if err := cmd.Start(); err != nil {
					log.Fatalf("Could not open file manager: %v", err)
				}
			}
			return
		}

		// On macOS, kardianos/service uses the deprecated `launchctl load/unload`
		// API which fails when running as root for system-level LaunchDaemons on
		// macOS Ventura+. Intercept start/stop/restart here and use the modern
		// `launchctl bootstrap/bootout` commands instead.
		// install and uninstall still go through kardianos (plist write/delete).
		if runtime.GOOS == "darwin" {
			plist := "/Library/LaunchDaemons/" + svcConfig.Name + ".plist"
			switch os.Args[1] {
			case "start":
				if err := darwinLaunchctl("bootstrap", "system", plist); err != nil {
					log.Fatalf("Service control failed (start): %v", err)
				}
				return
			case "stop":
				if err := darwinLaunchctl("bootout", "system", plist); err != nil {
					log.Printf("Service stop may have failed (not running?): %v", err)
				}
				return
			case "restart":
				bootstrapScript := "sleep 1 && launchctl bootstrap system " + plist + " &>/dev/null"
				_ = exec.Command("/bin/sh", "-c", bootstrapScript+" &").Run()
				_ = darwinLaunchctl("bootout", "system", plist)
				return
			case "uninstall":
				_ = darwinLaunchctl("bootout", "system", plist)
				// fall through to kardianos to remove the plist file
			}
		}

		if err := service.Control(svc, os.Args[1]); err != nil {
			log.Fatalf("Service control failed (%s): %v", os.Args[1], err)
		}
		return
	}

	//
	//
	// setup logging
	//

	var logWriter io.Writer
	
	// if interactive,
	// log to console
	if service.Interactive() {

		logWriter = os.Stdout

	// otherwise
	// log to file
	} else {

		logFile, err := os.OpenFile(pal.FileLog(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {

			// this will log to the system log on macOS and Linux
			// and the Windows Event Log on Windows
			// and then it will exit
			log.Fatalf("CRITICAL: Failed to open daemon log file %s: %v", pal.FileLog(), err)
		}

		logWriter = io.MultiWriter(os.Stdout, logFile)
	}

	slog.SetDefault(slog.New(logging.NewPlainHandler(slog.LevelDebug, logWriter)))

	//
	//
	// start program
	//
	slog.Debug("Starting spored daemon", "interactive", service.Interactive(), "os", runtime.GOOS, "arch", runtime.GOARCH)

	prg := &program{}

	// if interactive
	// running in console
	if service.Interactive() {

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		go prg.run()

		<-sigChan
		prg.stop()

		slog.Debug("Spored daemon stopped", "interactive", service.Interactive(), "os", runtime.GOOS, "arch", runtime.GOARCH)

	// otherwise
	// run as a service (kardianos/service)
	} else {

		svc, err := service.New(prg, svcConfig)
		if err != nil {
			slog.Error("Failed to create service", "error", err)
			os.Exit(1)
		}

		// No subcommand: run directly (blocks until signal, same as before).
		if err := svc.Run(); err != nil {
			slog.Error("Service error", "error", err)
			os.Exit(1)
		}
	}
}

// darwinLaunchctl runs a launchctl command and returns any error including
// the combined output so failures are easy to diagnose.
func darwinLaunchctl(args ...string) error {
	cmd := exec.Command("launchctl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl %v: %w\n%s", args, err, out)
	}
	return nil
}
