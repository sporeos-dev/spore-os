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

//
//
// private
// helper
//

var index atomic.Int64
func makeHandle() string {
	return fmt.Sprintf("hyphae-%d", index.Add(1))
}