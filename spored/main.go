package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"spored/internal/logging"
	"spored/internal/registry"

	"github.com/kardianos/service"
)

var svcConfig = &service.Config{
	Name:        "dev.spore.spored",
	DisplayName: "Spore OS Daemon",
	Description: "The Spore OS IPC hub daemon.",
	// UserName runs the daemon as the dedicated _spore system account.
	// This account is created by the installer and owns all spore data paths.
	// macOS: /Library/LaunchDaemons/ (requires sudo, runs at boot as _spore)
	// Linux: /etc/systemd/system/   (requires sudo, runs at boot as _spore)
	UserName: "_spore",
	Option: service.KeyValue{
		// kardianos/service defaults system daemon logs to /var/log which
		// _spore cannot write to. Use a directory the installer creates
		// with _spore ownership.
		"LogDirectory": "/Library/Logs/spore-os",
	},
}

func main() {
	slog.SetDefault(slog.New(logging.NewPlainHandler(slog.LevelDebug)))

	prg := &program{}
	svc, err := service.New(prg, svcConfig)
	if err != nil {
		slog.Error("Failed to create service", "error", err)
		os.Exit(1)
	}

	// Handle service control subcommands:
	//   spored install   — register with the OS service manager
	//   spored uninstall — deregister
	//   spored start     — start the background daemon
	//   spored stop      — stop the background daemon
	//   spored restart   — restart the background daemon
	if len(os.Args) > 1 {
		if os.Args[1] == "home" {
				var root string
				switch runtime.GOOS {
				case "darwin":
					root = "/Library/Application Support/spore-os"
				case "linux":
					root = "/var/lib/spore-os"
				case "windows":
					root = `C:\ProgramData\spore-os`
				default:
					root = "/tmp/spore-os"
				}
				var cmd *exec.Cmd
				switch runtime.GOOS {
				case "darwin":
					cmd = exec.Command("open", root)
				case "linux":
					cmd = exec.Command("xdg-open", root)
				case "windows":
					cmd = exec.Command("explorer", root)
				default:
					fmt.Println(root)
				}
				if cmd != nil {
					if err := cmd.Start(); err != nil {
						slog.Error("Could not open file manager", "error", err)
						os.Exit(1)
					}
			}
			return
		}

		// spored install <path>  — install a node manifest into the registry.
		// Distinguished from "spored install" (no path) which registers the
		// service with the OS service manager.
		if os.Args[1] == "install" && len(os.Args) > 2 {
			manifestPath := os.Args[2]
			r := &registry.Registry{}
			if err := r.Open(); err != nil {
				slog.Error("Failed to open registry", "error", err)
				os.Exit(1)
			}
			if err := r.Add(manifestPath); err != nil {
				slog.Error("Failed to install manifest", "path", manifestPath, "error", err)
				os.Exit(1)
			}
			slog.Info("Manifest installed — restart the daemon to load it", "path", manifestPath)
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
