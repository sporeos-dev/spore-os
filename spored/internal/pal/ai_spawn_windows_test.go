// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build windows

package pal

import (
	"os/exec"
	"testing"
)

func TestConfigureSpawnCommandHidesWindow(t *testing.T) {
	cmd := exec.Command("cmd.exe")
	ConfigureSpawnCommand(cmd)
	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.HideWindow {
		t.Fatal("spawn command does not hide its window")
	}
}
