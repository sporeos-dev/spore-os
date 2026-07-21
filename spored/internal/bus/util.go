package bus

import (
	"fmt"
	"time"
)

type witnessType string
const (
	incoming witnessType = "spore_incoming"
	outgoing witnessType = "spore_outgoing"
	event    witnessType = "spore_event"
	node     witnessType = "spore_node" 
)

func buildWitness(msg string, wtype witnessType, id string) string {
	time := time.Now().UnixMilli()
	idLabel := "from"
	if wtype == outgoing {
		idLabel = "to"
	}
	return fmt.Sprintf("witness %s spore_time=%d %s=%s %s", wtype, time, idLabel, id, msg)
}
