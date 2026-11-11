// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package hyphae

import (
	"fmt"
	"slices"
	"spored/internal/cparser"
	"spored/internal/event"
	sync_event "spored/internal/event/sync"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"spored/internal/witness"
	"strings"
	"sync/atomic"
)

func Confirm(node string, command string) (bool, bool) {
	
	// setup
	hcmd := "dev.sporeos.HYPHAE.confirm"
	handle := makeHandle()
	raw := fmt.Sprintf("%s ~%s node=%s command=%s", hcmd, handle, node, command)
	
	// send & receive
	sreq := sync_event.Request(raw, "dev.sporeos.SPORE")
	event.Dispatch().SynchronizedRequest.Notify(sreq)
	srep := sreq.Response()
	if srep == nil {
		witness.Error(error.Missing, error.Hyphae, "missing response")
		return false, false
	}

	// parse & return
	pm, ok := cparser.ParseAndValidate(srep.Message())
	if !ok {
		// witnessed in ParseAndValidate
		return false, false
	}

	// error/cancellation
	if slices.Contains(pm.Flags, "error") ||
	   slices.Contains(pm.Flags, "cancelled") {
		// witnessed in Incoming response
		return false, false

	// yes response
	} else if slices.Contains(pm.Flags, "yes") {
		if slices.Contains(pm.Flags, "no") {
			witness.Error(error.Collision, error.Hyphae, "confirmation returned 'yes' and 'no'")
			return false, false
		}
		return true, true
		
	// no response
	} else if slices.Contains(pm.Flags, "no") {
		return false, true
	}

	witness.Error(error.Malformed, error.Hyphae, "hyphae confirmation response failure")
	return false, false
}

func ManifestRead(path string) (string, bool) {

	// setup
	hcmd := "dev.sporeos.HYPHAE.manifest.read"
	handle := makeHandle()
	raw := fmt.Sprintf("%s ~%s path=%s", hcmd, handle, quotePath(path))

	// send & receive
	sreq := sync_event.Request(raw, "dev.sporeos.SPORE")
	event.Dispatch().SynchronizedRequest.Notify(sreq)
	srep := sreq.Response()
	if srep == nil {
		witness.Error(error.Missing, error.Hyphae, "missing response")
		return "", false
	}

	// parse & respond
	pm, ok := cparser.ParseAndValidate(srep.Message())
	if !ok {
		// witnessed in ParseAndValidate
		return "", false
	}

	if slices.Contains(pm.Flags, "error") || slices.Contains(pm.Flags, "cancelled") {
		// witnessed in Incoming response
		return "", false
	}

	content, ok := pm.Args["content"]
	if !ok {
		witness.Error(error.Missing, error.Hyphae, "missing in response", out.Pair("argument", "content"))
		return "", false
	}
	return content, true
}

func BinaryHash(pid int) (string, bool) {
	
	// setup
	hcmd := "dev.sporeos.HYPHAE.binary.hash"
	handle := makeHandle()
	raw := fmt.Sprintf("%s ~%s pid=%d", hcmd, handle, pid)

	// send & receive
	sreq := sync_event.Request(raw, "dev.sporeos.SPORE")
	event.Dispatch().SynchronizedRequest.Notify(sreq)
	srep := sreq.Response()
	if srep == nil {
		witness.Error(error.Missing, error.Hyphae, "missing response")
		return "", false
	}

	// parse & return
	pm, ok := cparser.ParseAndValidate(srep.Message())
	if !ok {
		// witnessed in ParseAndValidate
		return "", false
	}

	if slices.Contains(pm.Flags, "error") || slices.Contains(pm.Flags, "cancelled") {
		// witnessed in Incoming response
		return "", false
	}

	hash, ok := pm.Args["hash"]
	if !ok {
		witness.Error(error.Missing, error.Hyphae, "missing in response", out.Pair("argument", "hash"))
		return "", false
	}
	return hash, true
}

func FileHash(path string) (string, bool) {
	
	// setup
	hcmd := "dev.sporeos.HYPHAE.file.hash"
	handle := makeHandle()
	raw := fmt.Sprintf("%s ~%s path=%s", hcmd, handle, quotePath(path))

	// send & receive
	sreq := sync_event.Request(raw, "dev.sporeos.SPORE")
	event.Dispatch().SynchronizedRequest.Notify(sreq)
	srep := sreq.Response()
	if srep == nil {
		witness.Error(error.Missing, error.Hyphae, "missing response")
		return "", false
	}

	// parse & return
	pm, ok := cparser.ParseAndValidate(srep.Message())
	if !ok {
		// witnessed in ParseAndValidate
		return "", false
	}

	if slices.Contains(pm.Flags, "error") || slices.Contains(pm.Flags, "cancelled") {
		// witnessed in Incoming response
		return "", false
	}

	hash, ok := pm.Args["hash"]
	if !ok {
		witness.Error(error.Missing, error.Hyphae, "missing in response", out.Pair("argument", "hash"))
		return "", false
	}
	return hash, true
}

func Hspawn(path string) bool {

	// setup
	hcmd := "dev.sporeos.HYPHAE.hspawn"
	handle := makeHandle()
	raw := fmt.Sprintf("%s ~%s binary=%s", hcmd, handle, quotePath(path))

	// send & receive
	sreq := sync_event.Request(raw, "dev.sporeos.SPORE")
	event.Dispatch().SynchronizedRequest.Notify(sreq)
	srep := sreq.Response()
	if srep == nil {
		witness.Error(error.Missing, error.Hyphae, "missing response")
		return false
	}

	// parse & return
	pm, ok := cparser.ParseAndValidate(srep.Message())
	if !ok {
		// witnessed in ParseAndValidate
		return false
	}

	if slices.Contains(pm.Flags, "error") || slices.Contains(pm.Flags, "cancelled") {
		// witnessed in Incoming response
		return false
	}

	return true
}

func HKill(pid int) bool {

	// setup
	hcmd := "dev.sporeos.HYPHAE.hkill"
	handle := makeHandle()
	raw := fmt.Sprintf("%s ~%s pid=%d", hcmd, handle, pid)

	// send & receive
	sreq := sync_event.Request(raw, "dev.sporeos.SPORE")
	event.Dispatch().SynchronizedRequest.Notify(sreq)
	srep := sreq.Response()
	if srep == nil {
		witness.Error(error.Missing, error.Hyphae, "missing response")
		return false
	}

	// parse & return
	pm, ok := cparser.ParseAndValidate(srep.Message())
	if !ok {
		// witnessed in ParseAndValidate
		return false
	}

	if slices.Contains(pm.Flags, "error") || slices.Contains(pm.Flags, "cancelled") {
		// witnessed in Incoming response
		return false
	}
	
	return true
}

//
//
// private
// helper
//

var index atomic.Int64
func makeHandle() string {
	return fmt.Sprintf("hyphae-%d", index.Add(1))
}

func quotePath(path string) string {
	escaped := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		"\n", `\n`,
		"\r", `\r`,
		"\t", `\t`,
	).Replace(path)
	return `"` + escaped + `"`
}
