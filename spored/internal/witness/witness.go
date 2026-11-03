// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package witness

import (
	"log/slog"
	"spored/internal/iface"
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

