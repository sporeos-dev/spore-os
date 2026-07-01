// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package message

import (
	"encoding/json"
	"errors"
	"strings"
)

// Topic extends Message with a TopicName for pub/sub routing.
// Publish is the concrete implementation of Topic.
type Topic interface {
	Message
	TopicName() string
}

// Publish represents a pub/sub publish message on the wire.
// Wire format: publish <topic> [key=value ...]
// No handle — fire-and-forget. No response.
type Publish struct {
	raw       string
	source    string
	topicName string
	args      map[string]string
	flags     []string

	mid         int64
	destination string
}

func (p *Publish) IsCast() bool        { return false }
func (p *Publish) IsCapture() bool     { return false }
func (p *Publish) IsSpore() bool       { return false }
func (p *Publish) IsError() bool       { return false }
func (p *Publish) IsCustomError() bool { return false }
func (p *Publish) IsCancelled() bool   { return false }
func (p *Publish) IsPublish() bool     { return true }

func (p *Publish) Parse(raw string, from string) error {
	p.raw = raw
	p.source = from
	p.mid = -1
	p.args = make(map[string]string)
	p.flags = []string{}

	if !strings.HasPrefix(raw, "publish ") {
		return errors.New("publish message must start with 'publish '")
	}
	rest := strings.TrimPrefix(raw, "publish ")

	parts, err := Tokenize(rest)
	if err != nil {
		return err
	}
	if len(parts) == 0 {
		return errors.New("publish message missing topic name")
	}

	p.topicName = parts[0]

	for _, part := range parts[1:] {
		if strings.Contains(part, "=") {
			sub := strings.SplitN(part, "=", 2)
			p.args[sub[0]] = unquoteValue(sub[1])
		} else {
			p.flags = append(p.flags, part)
		}
	}

	return nil
}

// TopicName returns the topic this message was published to.
func (p *Publish) TopicName() string { return p.topicName }

// Source returns the publisher node ID.
func (p *Publish) Source() string { return p.source }

// Cast returns the publisher node ID (same as Source).
func (p *Publish) Cast() string { return p.source }

// Capture returns empty — publish messages have no response.
func (p *Publish) Capture() string { return "" }

// Command returns the topic name.
func (p *Publish) Command() string { return p.topicName }

// Handle returns empty — publish messages have no handle.
func (p *Publish) Handle() string { return "" }

func (p *Publish) ToString() string {
	// When delivering to a subscriber (destination set), the hub injects cast=
	// (publisher ID) and capture= (subscriber ID) per SPEC §10.2.
	if p.destination != "" {
		return p.raw + " cast=" + p.source + " capture=" + p.destination
	}
	return p.raw
}
func (p *Publish) SetMessageId(mid int64)    { p.mid = mid }
func (p *Publish) MessageId() int64          { return p.mid }
func (p *Publish) SetDestination(dst string) { p.destination = dst }
func (p *Publish) Destination() string       { return p.destination }

func (p *Publish) ToJsonString() string {
	result := map[string]interface{}{
		"topic":  p.topicName,
		"source": p.source,
		"args":   p.args,
		"flags":  p.flags,
	}
	data, err := json.Marshal(result)
	if err != nil {
		return "{}"
	}
	return string(data)
}
