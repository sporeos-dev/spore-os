package message

import (
	"fmt"
	"time"
)

type WitnessType string
const (
	WitnessIncoming WitnessType = "spore_incoming"
	WitnessOutgoing WitnessType = "spore_outgoing"
	WitnessSpore WitnessType = "spore_event"
	WitnessNode WitnessType = "spore_node"
)

type witness struct {
	raw string
}

func Witness(wtype WitnessType, body string, id string) Message {
	t := time.Now().UnixMilli()
	var raw string
	switch wtype {
	case WitnessNode:
		raw = fmt.Sprintf("witness %s cast=%s %s spore_time=%d", body, id, wtype, t)
	case WitnessIncoming:
		raw = fmt.Sprintf("witness %s %s spore_time=%d", body, wtype, t)
	case WitnessOutgoing:
		raw = fmt.Sprintf("witness %s %s spore_time=%d", body, wtype, t)
	case WitnessSpore:
		raw = fmt.Sprintf("witness %s %s spore_time=%d", body, wtype, t)
	}
	return &witness{
		raw: raw,
	}
}

func (w *witness) Get() string {
	return w.raw
}
