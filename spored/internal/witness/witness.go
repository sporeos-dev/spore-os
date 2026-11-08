// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package witness

import (
	"log/slog"
	"spored/internal/event"
	witness_event "spored/internal/event/witness"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
)

func Incoming(msg, orig string) {
	event.Dispatch().Witness.Notify(witness_event.Incoming(msg, orig))
}

func Outgoing(msg, orig string) {
	event.Dispatch().Witness.Notify(witness_event.Outgoing(msg, orig))
}

func Spore(msg string, outs ...out.IOut) {
	event.Dispatch().Witness.Notify(witness_event.Spore(msg, outs...))
}

func Node(msg, orig string) {
	event.Dispatch().Witness.Notify(witness_event.Node(msg, orig))
}

func Error(code error.Code, module error.Module, what string, outs ...out.IOut) {
	ctx := witness_event.Error(code, module, what, outs...)
	slog.Error(ctx.Message())
	event.Dispatch().Witness.Notify(ctx)
}
