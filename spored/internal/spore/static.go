package spore

import (
	"fmt"
	"slices"
	"spored/internal/cparser"
	"spored/internal/event"
	sync_event "spored/internal/event/sync"
	"spored/internal/utilities/error"
	"spored/internal/witness"
	"sync/atomic"
)

func NodeSpawn(node string) bool {

	// setup
	cmd := "dev.sporeos.SPORE.node.spawn"
	handle := makeHandle()
	raw := fmt.Sprintf("%s ~%s node=%s", cmd, handle, node)

	// send & receive
	sreq := sync_event.Request(raw, "dev.sporeos.SPORE")
	event.Dispatch().SynchronizedRequest.Notify(sreq)
	srep := sreq.Response()
	if srep == nil {
		witness.Error(error.Missing, error.Spore, "missing response")
		return false
	}

	pm, ok := cparser.ParseAndValidate(srep.Message())
	if !ok {
		// witnessed in ParseAndValidate
		return false
	}

	// error/cancellation
	if slices.Contains(pm.Flags, "error") ||
	   slices.Contains(pm.Flags, "cancel") {
		// witnessed in Incoming response
		return false
	}

	return true
}

func StatePersist(key string, node string, body string) bool {

	// setup
	cmd := "dev.sporeos.SPORE.state.persist"
	handle := makeHandle()
	raw := fmt.Sprintf("%s ~%s key=%s body=%s", cmd, handle, key, body)

	// send & receive
	sreq := sync_event.Request(raw, "dev.sporeos.SPORE")
	event.Dispatch().SynchronizedRequest.Notify(sreq)
	srep := sreq.Response()
	if srep == nil {
		witness.Error(error.Missing, error.Spore, "missing response")
		return false
	}

	pm, ok := cparser.ParseAndValidate(srep.Message())
	if !ok {
		// witnessed in ParseAndValidate
		return false
	}

	if slices.Contains(pm.Flags, "error") ||
	   slices.Contains(pm.Flags, "cancel") {
		witness.Error(error.Generic, error.Spore, "error in response")
		return false
	}

	return true
}

func StateRetrieve(key string, node string) (string, bool) {

	// setup
	cmd := "dev.sporeos.SPORE.state.retrieve"
	handle := makeHandle()
	raw := fmt.Sprintf("%s ~%s key=%s", cmd, handle, key)

	// send & receive
	sreq := sync_event.Request(raw, "dev.sporeos.SPORE")
	event.Dispatch().SynchronizedRequest.Notify(sreq)
	srep := sreq.Response()
	if srep == nil {
		witness.Error(error.Missing, error.Spore, "missing response")
		return "", false
	}

	pm, ok := cparser.ParseAndValidate(srep.Message())
	if !ok {
		// witnessed in ParseAndValidate
		return "", false
	}

	if slices.Contains(pm.Flags, "error") ||
	   slices.Contains(pm.Flags, "cancelled") {
		witness.Error(error.Generic, error.Spore, "error in response")
		return "", false
	}

	body, ok := pm.Args["body"]
	if !ok {
		witness.Error(error.Missing, error.Spore, "missing body")
		return "", false
	}

	return body, true
}

func RequestPermission(node, cap string) (string, bool) {

	// setup
	cmd := "dev.sporeos.SPORE.permission.request"
	handle := makeHandle()
	raw := fmt.Sprintf("%s ~%s capability=%s", cmd, handle, cap)

	// send & receive
	sreq := sync_event.Request(raw, fmt.Sprintf("dev.sporeos.SPORE:%s", node))
	event.Dispatch().SynchronizedRequest.Notify(sreq)
	srep := sreq.Response()
	if srep == nil {
		witness.Error(error.Missing, error.Spore, "missing response")
		return "", false
	}

	pm, ok := cparser.ParseAndValidate(srep.Message())
	if !ok {
		// witnessed in ParseAndValidate
		return "", false
	}

	if slices.Contains(pm.Flags, "error") ||
	   slices.Contains(pm.Flags, "cancelled") {
		witness.Error(error.Generic, error.Spore, "cancelled")
		return "", false
	}

	if slices.Contains(pm.Flags, "no") {
		witness.Spore("permission denied")
		return "", false
	}

	token, ok := pm.Args["permission.token"]
	if !ok {
		witness.Error(error.Missing, error.Spore, "missing token")
		return "", false
	}

	return token, true
}

//
//
// private
// helper
//

var index atomic.Int64
func makeHandle() string {
	return fmt.Sprintf("spore-%d", index.Add(1))
}