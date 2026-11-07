// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package message

import (
	"fmt"
	"spored/internal/iface"
	"spored/internal/utilities/out"
	"strings"
	"time"
)

type witness struct {
	raw         string
	cast        string
	witnessFlag string
}

// from spore
func Witness(body string, outs ...out.IOut) iface.Message {
	witnessFlag := "spore.event"
	var builder strings.Builder
	builder.WriteString(body)
	for _, el := range outs {
		if el.String() == "spore.incoming" || el.String() == "spore.outgoing" || el.String() == "spore.node" {
			witnessFlag = el.String()
			continue
		}
		builder.WriteByte(' ')
		builder.WriteString(el.String())
	}
	return &witness{
		raw:         builder.String(),
		cast:        "n/a",
		witnessFlag: witnessFlag,
	}
}

// from node
func Node(raw string, cast string) iface.Message {
	raw = strings.ReplaceAll(raw, "\n", "\\n")
	raw = strings.ReplaceAll(raw, "\"", "\\\"")
	return &witness{
		raw:         raw,
		cast:        cast,
		witnessFlag: "spore.node",
	}
}

// incoming
func Incoming(raw string, cast string) iface.Message {
	raw = strings.ReplaceAll(raw, "\n", "\\n")
	raw = strings.ReplaceAll(raw, "\"", "\\\"")
	return &witness{
		raw:         raw,
		cast:        cast,
		witnessFlag: "spore.incoming",
	}
}

// outgoing
func Outgoing(raw string, cast string) iface.Message {
	raw = strings.ReplaceAll(raw, "\n", "\\n")
	raw = strings.ReplaceAll(raw, "\"", "\\\"")
	return &witness{
		raw:         raw,
		cast:        cast,
		witnessFlag: "spore.outgoing",
	}
}

func (w *witness) Capability() string {
	return "n/a"
}

func (w *witness) Cast() string {
	return w.cast
}

func (w *witness) Actor() string {
	return castIdentity(w.cast).Actor
}

func (w *witness) Origin() string {
	return castIdentity(w.cast).Origin
}

func (w *witness) Capture() string {
	return "n/a"
}

func (w *witness) Arg(key string) (string, bool) {
	return "", false
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

// func (w *witness) ToJSON() string {
// 	return w.raw
// }

func (w *witness) Wire() string {
	return "n/a"
}

// raw
// --> Could be anything.
// witness spore: +witness +message +spore.event +spore.time
// --> witness body='<raw>' spore.event spore.time=time
// witness node: +witness +body +cast +spore.node +spore.time
// --> witness body='<raw>' cast=witnesser.id spore.node spore.time=time
func (w *witness) Witness() string {
	t := time.Now().UnixMilli()

	// spore.event
	if w.cast == "n/a" {
		return fmt.Sprintf(`witness body='%s' %s spore.time=%d`, w.raw, w.witnessFlag, t)

		// spore.node
	} else {
		return fmt.Sprintf(`witness body='%s' cast=%s %s spore.time=%d`, w.raw, w.cast, w.witnessFlag, t)
	}
}
