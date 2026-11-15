// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build !windows

package pal

import "os/exec"

func ConfigureSpawnCommand(cmd *exec.Cmd) {}
