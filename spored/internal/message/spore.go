// Copyright 2026 mharr
// SPDX-License-Identifier: AGPL-3.0-only

package message

import (
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
)

type Spore struct {
	raw string
	cast string
	capture string
	command string
	handle string

	args map[string]string
	flags map[string]bool

	mid int64
}

func (s* Spore) IsCast() bool        { return false }
func (s* Spore) IsCapture() bool     { return false }
func (s* Spore) IsSpore() bool       { return true }
func (s* Spore) IsError() bool       { return false }
func (s* Spore) IsCustomError() bool { return false }
func (s* Spore) IsCancelled() bool   { return false }

func (s* Spore) Parse(raw string, from string) error {
	s.raw = raw
	s.cast = from
	s.capture = "dev.sporeos.SPORE"
	s.args = make(map[string]string)
	s.flags = make(map[string]bool)
	s.mid = -1

	parts, err := Tokenize(raw)
	if err != nil {
		return err
	}
	if len(parts) == 0 {
		return nil
	}

	// First part is the command (subject)
	s.command = parts[0]

	// Parse all tokens after the command: handle, args, and flags may appear in any order.
	for _, part := range parts[1:] {
		if strings.HasPrefix(part, "~") {
			s.handle = strings.TrimPrefix(part, "~")
		} else if strings.Contains(part, "=") {
			sub := strings.SplitN(part, "=", 2)
			s.args[sub[0]] = unquoteValue(sub[1])
		} else {
			s.flags[part] = true
		}
	}

	if s.handle == "" {
		return errors.New("handle (~xyz) required")
	}

	return nil
}

func (s* Spore) Cast() string {
	return s.cast
}

func (s* Spore) Capture() string {
	return s.capture
}

func (s* Spore) Command() string {
	return s.command
}

func (s* Spore) Handle() string {
	return s.handle
}

func (s* Spore) ToString() string {
	return s.raw
}

func (s* Spore) ToJsonString() string {
	flags := make([]string, 0, len(s.flags))
	for f := range s.flags {
		flags = append(flags, f)
	}
	result := map[string]interface{}{
		"command": s.command,
		"handle":  s.handle,
		"args":    s.args,
		"flags":   flags,
	}
	data, err := json.Marshal(result)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func (s* Spore) SetMessageId(mid int64) {
	s.mid = mid
}

func (s* Spore) MessageId() int64 {
	return s.mid
}

func (s* Spore) Source() string {
	return s.cast
}

func (s* Spore) SetDestination(dst string) {
	slog.Warn("Cannot set destination on SPORE messages")
}

func (s* Spore) Destination() string {
	return s.capture
}

func (s* Spore) HasFlag(flag string) bool {
	return s.flags[flag]
}

func (s* Spore) GetArgument(arg string) (string, error) {
	val, exists := s.args[arg]
	if !exists {
		return "", errors.New("missing argument " + arg)
	}
	return val, nil
}

func (s* Spore) GetArgumentIf(arg string, def string) string {
	val, exists := s.args[arg]
	if !exists {
		return def
	}
	return val
}

// HasInlines reports whether this SPORE message contains any inline call forms.
func (s *Spore) HasInlines() bool {
	parts, err := Tokenize(s.raw)
	if err != nil {
		return false
	}
	p := ParseArgs(parts)
	return p.HasInlineForm()
}

// GetParsedArgs returns the structured argument representation of this SPORE message.
func (s *Spore) GetParsedArgs() (ParsedArgs, error) {
	parts, err := Tokenize(s.raw)
	if err != nil {
		return ParsedArgs{}, err
	}
	return ParseArgs(parts), nil
}