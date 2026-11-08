// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package witness

import (
	"log/slog"
	"spored/internal/event"
	witness_event "spored/internal/event/witness"
	"spored/internal/iface"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
)

type ibus interface {
	Witness(msg iface.Message)
}

var messageChan = make(chan iface.Message, 100)

func Init(bus ibus) {
	go func() {
		for msg := range messageChan {
			bus.Witness(msg)
		}
	}()
}

func Send(msg iface.Message) {
	select {
	case messageChan <- msg:
	default:
		slog.Warn("witness channel full, dropping next message")
	}
}

func Incoming(msg, orig string) {
	event.Dispatch().Witness.Notify(witness_event.Incoming(msg, orig))
}

func Outgoing(msg, orig string) {
	event.Dispatch().Witness.Notify(witness_event.Outgoing(msg, orig))
}

func Spore(msg string) {
	event.Dispatch().Witness.Notify(witness_event.Spore(msg))
}

func Node(msg, orig string) {
	event.Dispatch().Witness.Notify(witness_event.Node(msg, orig))
}

func Error(code error.Code, module error.Module, what string, outs []out.IOut) {
	event.Dispatch().Witness.Notify(witness_event.Error(code, module, what, outs))
}

