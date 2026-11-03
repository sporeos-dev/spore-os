// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package loc

import (
	"runtime"
	"strconv"
)

func Print() {
	_, f, l, ok := runtime.Caller(1) 
	if !ok {
		println("unknown: -1")
		// fmt.Println("%s: %d", "unknown", -1)
		return
	}
	println(f + ": " + strconv.Itoa(l))
	// fmt.Println("%s: %d", f, l)
}   