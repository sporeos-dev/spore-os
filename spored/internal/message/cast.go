// Copyright 2026 mharr
// SPDX-License-Identifier: AGPL-3.0-only

package message

import (
	"encoding/json"
	"errors"
	"strings"
)

type Cast struct {
	raw         string
	originalRaw string // raw before hub-injected cast= is appended
	cast        string
	command     string
	handle      string

	mid         int64
	destination string
}

func (c* Cast) IsCast() bool        { return true }
func (c* Cast) IsCapture() bool     { return false }
func (c* Cast) IsSpore() bool       { return false }
func (c* Cast) IsError() bool       { return false }
func (c* Cast) IsCustomError() bool { return false }
func (c* Cast) IsCancelled() bool   { return false }

func (c* Cast) Parse(raw string, from string) error {
	c.raw = raw
	c.cast = from
	c.mid = -1

	parts, err := Tokenize(raw)
	if err != nil {
		return err
	}
	if len(parts) == 0 {
		return errors.New("cast message is empty")
	}

	// First part is the command (subject)
	c.command = parts[0]

	// Look for handle (starts with ~) and parse hub-injected cast= if present
	for _, part := range parts[1:] {
		if strings.HasPrefix(part, "~") {
			c.handle = strings.TrimPrefix(part, "~")
		} else if strings.HasPrefix(part, "cast=") {
			c.cast = strings.TrimPrefix(part, "cast=")
		}
	}

	if c.handle == "" {
		return errors.New("handle (~xyz) required")
	}

	// Store original raw before injection so ParsedArgs can reconstruct the
	// message without the hub-injected cast= field.
	c.originalRaw = raw
	// Inject cast= so receivers see the originating node ID.
	c.raw = raw + " cast=" + c.cast
	return nil
}

// HasInlines reports whether this cast message contains any inline call forms
// (key-binding or spread). The hub uses this as a fast pre-routing check.
func (c *Cast) HasInlines() bool {
	parts, err := Tokenize(c.originalRaw)
	if err != nil {
		return false
	}
	p := ParseArgs(parts)
	return p.HasInlineForm()
}

// GetParsedArgs returns the structured argument representation of this cast,
// built from the pre-injection raw (without the hub-appended cast= field).
// This is what the inline resolver should use as the held outer message so
// that Rebuild() produces a clean message string ready for re-routing.
func (c *Cast) GetParsedArgs() (ParsedArgs, error) {
	parts, err := Tokenize(c.originalRaw)
	if err != nil {
		return ParsedArgs{}, err
	}
	return ParseArgs(parts), nil
}

func (c* Cast) Cast() string {
	return c.cast
}

func (c* Cast) Capture() string {
	return ""
}

func (c* Cast) Command() string {
	return c.command
}

func (c* Cast) Handle() string {
	return c.handle
}

func (c* Cast) ToString() string {
	return c.raw
}

func (c* Cast) HasFlag(flag string) bool {
	parts, err := Tokenize(c.raw)
	if err != nil || len(parts) < 2 {
		return false
	}
	for _, part := range parts[1:] {
		if part == flag {
			return true
		}
	}
	return false
}

func (c* Cast) ToJsonString() string {
	args := make(map[string]string)
	flags := []string{}

	parts, _ := Tokenize(c.raw)
	for _, part := range parts[1:] {
		if strings.HasPrefix(part, "~") {
			// skip handle
		} else if strings.Contains(part, "=") {
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

func (c* Cast) SetMessageId(mid int64) {
	c.mid = mid
}

func (c* Cast) MessageId() int64 {
	return c.mid
}

func (c* Cast) Source() string {
	return c.cast
}

func (c* Cast) SetDestination(dst string) {
	c.destination = dst
}

func (c* Cast) Destination() string {
	return c.destination
}

