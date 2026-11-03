// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package error

import "spored/internal/utilities/out"

func MissingArg(arg string, mod Module) *Error {
	return New(
		Missing,
		mod,
		"missing argument",
		out.Pair("argument", arg))
}