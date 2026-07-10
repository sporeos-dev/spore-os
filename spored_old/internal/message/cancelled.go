// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package message

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Cancelled represents a cancelled response — a graceful non-result where the
// call completed without error but produced no output (e.g. the user dismissed
// a dialog).
//
// On the wire, the responding node writes:
//
//	~handle:subject cancelled
//
// and the hub injects capture= before forwarding to the caller, producing:
//
//	~handle:subject cancelled capture=node_id
//
// Exactly one of ok, error, custom_error, or cancelled is present on every
// response; cancelled and ok are mutually exclusive.
type Cancelled struct {
	raw     string
	cast    string
	capture string
	command string
	handle  string

	mid         int64
	destination string
}

func (c *Cancelled) IsCast() bool        { return false }
func (c *Cancelled) IsCapture() bool     { return false }
func (c *Cancelled) IsSpore() bool       { return false }
func (c *Cancelled) IsError() bool       { return false }
func (c *Cancelled) IsCustomError() bool { return false }
func (c *Cancelled) IsCancelled() bool   { return true }
func (c *Cancelled) IsPublish() bool     { return false }

func (c *Cancelled) Parse(raw string, from string) error {
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

	// First token is ~handle:command
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

	for _, part := range parts[1:] {
		if strings.HasPrefix(part, "cast=") {
			c.cast = strings.TrimPrefix(part, "cast=")
		} else if strings.HasPrefix(part, "capture=") {
			c.capture = strings.TrimPrefix(part, "capture=")
		}
	}

	// Inject capture= so the caller knows which node cancelled.
	// Unlike Capture, we do NOT inject ok — cancelled is not a success.
	c.raw = raw + " capture=" + c.capture
	return nil
}

// NewCancelled builds a hub-generated cancelled response for the given original
// outer call. Used by the inline resolver when an inner call returns cancelled
// and the hub must propagate it back to the original caller.
func NewCancelled(original Message) *Cancelled {
	c := &Cancelled{}
	c.handle = original.Handle()
	c.command = original.Command()
	c.capture = "SPORE.hub"
	c.cast = original.Cast()
	c.mid = -1
	c.raw = fmt.Sprintf("~%s:%s cancelled capture=SPORE.hub", c.handle, c.command)
	return c
}

func (c *Cancelled) Cast() string    { return c.cast }
func (c *Cancelled) Capture() string { return c.capture }
func (c *Cancelled) Command() string { return c.command }
func (c *Cancelled) Handle() string  { return c.handle }
func (c *Cancelled) ToString() string { return c.raw }

func (c *Cancelled) ToJsonString() string {
	result := map[string]interface{}{
		"command":   c.command,
		"handle":    c.handle,
		"capture":   c.capture,
		"cancelled": true,
	}
	data, err := json.Marshal(result)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func (c *Cancelled) SetMessageId(mid int64)    { c.mid = mid }
func (c *Cancelled) MessageId() int64          { return c.mid }
func (c *Cancelled) Source() string            { return c.capture }
func (c *Cancelled) SetDestination(dst string) { c.destination = dst }
func (c *Cancelled) Destination() string       { return c.destination }
