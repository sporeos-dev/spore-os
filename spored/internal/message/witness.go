package message

import (
	"fmt"
	"spored/internal/utilities/error"
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

func (w *witness) IsWitness() bool {
	return true
}

func (w *witness) Command() string {
	return "n/a"
}

func (w *witness) Cast() string {
	return "n/a"
}

func (w *witness) Arg(key string) (string, *error.Error) {
	return "", error.New(
		error.NotApplicable,
		error.Message,
		"witness messages do not support arguments",
		nil)
}

func (w *witness) ArgIf(key string, ifnot string) string {
	return ifnot
}

func (w *witness) Flag(flag string) bool {
	return false
}

func (w *witness) Handle() string {
	return "n/a"
}

func (w *witness) ToJSON() string {
	return w.raw
}

func (w *witness) Topic() string {
	return "n/a"
}