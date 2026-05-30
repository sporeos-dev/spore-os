// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package message

import (
	"encoding/json"
	"strings"
)

type Capture struct {
	raw string
	cast string
	capture string
	command string
	handle string

	mid int64
	destination string
}

func (c* Capture) IsCast() bool        { return false }
func (c* Capture) IsCapture() bool     { return true }
func (c* Capture) IsSpore() bool       { return false }
func (c* Capture) IsError() bool       { return false }
func (c* Capture) IsCustomError() bool { return false }
func (c* Capture) IsCancelled() bool   { return false }

func (c* Capture) Parse(raw string, from string) error {
	c.raw = raw
	c.capture = from
	c.mid = -1

	parts, err := Tokenize(raw)
	if err != nil {
		return err
	}
	if len(parts) == 0 {
		return nil
	}

	// First part is ~handle:command format
	first := parts[0]
	if strings.HasPrefix(first, "~") {
		handleCommand := strings.TrimPrefix(first, "~")
		handleParts := strings.SplitN(handleCommand, ":", 2)
		if len(handleParts) >= 1 {
			c.handle = handleParts[0]
		}
		if len(handleParts) >= 2 {
			c.command = handleParts[1]
		}
	}

	// Parse hub-injected cast= and capture= fields if present
	for _, part := range parts[1:] {
		if strings.HasPrefix(part, "cast=") {
			c.cast = strings.TrimPrefix(part, "cast=")
		} else if strings.HasPrefix(part, "capture=") {
			c.capture = strings.TrimPrefix(part, "capture=")
		}
	}

	// Inject ok and capture= so the caster sees who responded.
	c.raw = raw + " ok capture=" + c.capture
	return nil
}

func (c* Capture) Cast() string {
	return c.cast
}

func (c* Capture) Capture() string {
	return c.capture
}

func (c* Capture) Command() string {
	return c.command
}

func (c* Capture) Handle() string {
	return c.handle
}

func (c* Capture) ToString() string {
	return c.raw
}

func (c* Capture) ToJsonString() string {
	args := make(map[string]string)
	flags := []string{}

	parts, _ := Tokenize(c.raw)
	for _, part := range parts[1:] {
		if strings.Contains(part, "=") {
			sub := strings.SplitN(part, "=", 2)
			args[sub[0]] = unquoteValue(sub[1])
		} else {
			flags = append(flags, part)
		}
	}

	result := map[string]interface{}{
		"command": c.command,
		"handle":  c.handle,
		"args":    args,
		"flags":   flags,
	}
	data, err := json.Marshal(result)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func (c* Capture) SetMessageId(mid int64) {
	c.mid = mid
}

func (c* Capture) MessageId() int64 {
	return c.mid
}

func (c* Capture) Source() string {
	return c.capture
}

func (c* Capture) SetDestination(dst string) {
	c.destination = dst
}

func (c* Capture) Destination() string {
	return c.destination
}
