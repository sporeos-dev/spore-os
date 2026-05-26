package message

import (
	"errors"
	"strings"
)

type Message interface {
	IsCast() bool
	IsCapture() bool
	IsSpore() bool
	IsError() bool
	IsCustomError() bool
	IsCancelled() bool

	Parse(raw string, from string) error
	Cast() string
	Capture() string
	Command() string
	Handle() string
	ToString() string
	ToJsonString() string

	SetMessageId(mid int64)
	MessageId() int64
	Source() string
	SetDestination(dst string)
	Destination() string
}

func Parse(raw string, from string) (Message, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("empty")
	}

	var msg Message
	if strings.HasPrefix(raw, "SPORE.") {
		spore := &Spore{}
		msg = spore
	} else if strings.HasPrefix(raw, "~") {
		if containsErrorFlag(raw) {
			errMsg := &NodeError{}
			msg = errMsg
		} else if containsCancelledFlag(raw) {
			cancelled := &Cancelled{}
			msg = cancelled
		} else {
			capture := &Capture{}
			msg = capture
		}
	} else {
		cast := &Cast{}
		msg = cast
	}

	err := msg.Parse(raw, from)
	if err != nil {
		return nil, err
	}

	return msg, err
}

// ExtractHandle attempts to find a ~handle token in a raw message string.
// Returns empty string if not found. Used when parse fails and the message
// type is unknown but a handle may still be present for error feedback.
func ExtractHandle(raw string) string {
	parts, err := Tokenize(raw)
	if err != nil {
		parts = strings.Fields(raw) // best-effort fallback on malformed input
	}
	for _, part := range parts {
		if strings.HasPrefix(part, "~") {
			token := strings.TrimPrefix(part, "~")
			// Strip any :subject suffix (capture/error shape)
			if idx := strings.IndexByte(token, ':'); idx >= 0 {
				token = token[:idx]
			}
			return token
		}
	}
	return ""
}

func containsCancelledFlag(raw string) bool {
	parts, err := Tokenize(raw)
	if err != nil {
		parts = strings.Fields(raw) // best-effort fallback on malformed input
	}
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts[1:] {
		if part == "cancelled" {
			return true
		}
	}
	return false
}

func containsErrorFlag(raw string) bool {
	parts, err := Tokenize(raw)
	if err != nil {
		parts = strings.Fields(raw) // best-effort fallback on malformed input
	}
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts[1:] {
		if part == "error" || part == "custom_error" {
			return true
		}
	}
	return false
}
