// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package iface

type Message interface {
	Capability() string
	Cast() string
	Capture() string
	Arg(key string) (string, bool)
	ArgIf(key string, ifnot string) string
	Flag(flag string) bool
	Handle() string

	Wire() string
	Witness() string
}
