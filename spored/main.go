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
	slog.Info("Starting spored daemon", "interactive", service.Interactive(), "os", runtime.GOOS, "arch", runtime.GOARCH)

	prg := &program{}

	// if interactive
	// running in console
	if service.Interactive() {

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		go prg.run()

		<-sigChan
		prg.stop()

		slog.Info("Spored daemon stopped", "interactive", service.Interactive(), "os", runtime.GOOS, "arch", runtime.GOARCH)

	// otherwise
	// run as a service (kardianos/service)
	} else {

		svc, err := service.New(prg, svcConfig)
		if err != nil {
			slog.Error("Failed to create service", "error", err)
			os.Exit(1)
		}

		// Handle service control subcommands:
		//   spored home      — open the file manager to the root data directory
		//   spored install   — register with the OS service manager
		//   spored uninstall — deregister
		//   spored start     — start the background daemon
		//   spored stop      — stop the background daemon
		//   spored restart   — restart the background daemon
		if len(os.Args) > 1 {
			if os.Args[1] == "home" {
					root := pal.DirectoryRoot()
					open := pal.CommandOpenFileManager()
					cmd := exec.Command(open, root)
					if cmd != nil {
						if err := cmd.Start(); err != nil {
							slog.Error("Could not open file manager", "error", err)
							os.Exit(1)
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
						slog.Error("Service control failed", "action", "start", "error", err)
						os.Exit(1)
					}
					return
				case "stop":
					// bootout returns an error if not running; treat that as a warning.
					if err := darwinLaunchctl("bootout", "system", plist); err != nil {
						slog.Warn("Service stop may have failed (not running?)", "error", err)
					}
					return
				case "restart":
					// bootout sends SIGKILL to this process before it can run bootstrap.
					// Schedule bootstrap in a detached background shell first — the shell
					// outlives us and runs bootstrap after the old daemon has been torn down.
					bootstrapScript := "sleep 1 && launchctl bootstrap system " + plist + " &>/dev/null"
					_ = exec.Command("/bin/sh", "-c", bootstrapScript+" &").Run()
					_ = darwinLaunchctl("bootout", "system", plist) // ignore not-running error
					return
				case "uninstall":
					// Ensure the daemon is stopped before kardianos deletes the plist.
					_ = darwinLaunchctl("bootout", "system", plist)
					// fall through to kardianos to remove the plist file
				}
			}

			if err := service.Control(svc, os.Args[1]); err != nil {
				slog.Error("Service control failed", "action", os.Args[1], "error", err)
				os.Exit(1)
			}
			return
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
