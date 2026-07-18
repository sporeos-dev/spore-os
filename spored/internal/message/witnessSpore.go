package message

import (
	"fmt"
	"time"
)

type sporeWitness struct {
	message string
}

func Witness(t Type, msg string, n inode) Message {
	time := time.Now().UnixMilli()
	var msgOut string

	switch t {
	case Incoming:
		msgOut = fmt.Sprintf("witness %s spore_incoming spore_time=%d", msg, time)
	case Outgoing:
		msgOut = fmt.Sprintf("witness %s spore_outgoing spore_time=%d", msg, time)
	case Spore:
		msgOut = fmt.Sprintf("witness %s spore_event spore_time=%d", msg, time)
	case Node:
		var id string
		if n == nil {
			id = "unknown_id"
		} else {
			id = n.Id()
		}
		msgOut = fmt.Sprintf("witness %s cast=%s spore_node spore_time=%d", msg, id, time)
	default:
		msgOut = fmt.Sprintf("witness %s spore_unknown spore_time=%d", msg, time)
	}

	return &sporeWitness{message: msgOut}
}

func (w *sporeWitness) IsWitness() bool {
	return true
}

func (w *sporeWitness) Output() string {
	return w.message
}

func (w *sporeWitness) Topic() string { return "" }
