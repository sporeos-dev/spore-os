// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package message

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ErrorCode string
const (
	// Internal / meta
	ErrorCodeUnknownFailure  ErrorCode = "UnknownFailure"
	ErrorCodeSporeFailure    ErrorCode = "SporeFailure"
	ErrorCodeProtocolFailure ErrorCode = "ProtocolFailure"
	ErrorCodeConnectionFailure ErrorCode = "ConnectionFailure"

	// General
	ErrorCodeGeneric            ErrorCode = "Generic"
	ErrorCodeFatal              ErrorCode = "Fatal"
	ErrorCodeTimeout            ErrorCode = "Timeout"
	ErrorCodeBusy               ErrorCode = "Busy"
	ErrorCodeResourcesExhausted ErrorCode = "ResourcesExhausted"
	ErrorCodeDeprecated         ErrorCode = "Deprecated"
	ErrorCodeRuntime            ErrorCode = "Runtime"
	ErrorCodeLogic              ErrorCode = "Logic"
	ErrorCodeReservedKeyword    ErrorCode = "ReservedKeyword"

	// Route
	ErrorCodeRouteNotFound       ErrorCode = "RouteNotFound"
	ErrorCodeRouteNotConnected   ErrorCode = "RouteNotConnected"
	ErrorCodeRouteNotAvailable   ErrorCode = "RouteNotAvailable"
	ErrorCodeRouteNotAllowed     ErrorCode = "RouteNotAllowed"
	ErrorCodeRouteNotImplemented ErrorCode = "RouteNotImplemented"

	// Message
	ErrorCodeMessageNotValid  ErrorCode = "MessageNotValid"
	ErrorCodeMessageMalformed ErrorCode = "MessageMalformed"

	// Arguments
	ErrorCodeArgumentMissing      ErrorCode = "RequiredArgumentMissing"
	ErrorCodeArgumentInvalidType  ErrorCode = "ArgumentInvalidType"
	ErrorCodeArgumentConflict     ErrorCode = "ArgumentConflict"
	ErrorCodeArgumentOutOfRange   ErrorCode = "ArgumentOutOfRange"
	ErrorCodeArgumentUnrecognized ErrorCode = "ArgumentUnrecognized"
	ErrorCodeArgumentDuplicated   ErrorCode = "ArgumentDuplicated"

	// Flags
	ErrorCodeFlagConflict     ErrorCode = "FlagConflict"
	ErrorCodeFlagUnrecognized ErrorCode = "FlagUnrecognized"
	ErrorCodeFlagDuplicated   ErrorCode = "FlagDuplicated"

	// Handles
	ErrorCodeHandleMissing ErrorCode = "HandleMissing"
	ErrorCodeHandleInUse   ErrorCode = "HandleInUse"
	ErrorCodeHandleExpired ErrorCode = "HandleExpired"
)

// ErrorInfo describes a single standard error code.
type ErrorInfo struct {
	Code        ErrorCode
	Description string
	Examples    []string
}

// StandardErrors returns all standard protocol error codes with descriptions and examples.
func StandardErrors() []ErrorInfo {
	return []ErrorInfo{
		// Internal / meta
		{ErrorCodeUnknownFailure,    "An unclassified or unknown error occurred.", nil},
		{ErrorCodeSporeFailure,      "An internal hub/runtime failure. Hub-only — nodes cannot emit this code.", nil},
		{ErrorCodeProtocolFailure,   "The hub encountered a protocol-level failure.", nil},
		{ErrorCodeConnectionFailure, "A connection error occurred between nodes.", nil},
		// General
		{ErrorCodeGeneric,            "A generic error with no further classification.", nil},
		{ErrorCodeFatal,              "A fatal, unrecoverable error.", nil},
		{ErrorCodeTimeout,            "The operation timed out before completing.", nil},
		{ErrorCodeBusy,               "The node or resource is currently busy.", nil},
		{ErrorCodeResourcesExhausted, "Resources (memory, handles, etc.) are exhausted.", nil},
		{ErrorCodeDeprecated,         "The command or feature is deprecated.", nil},
		{ErrorCodeRuntime,            "A runtime error occurred during execution.", nil},
		{ErrorCodeLogic,              "A logic error in the node's internal processing.", nil},
		// Route
		{ErrorCodeRouteNotFound,       "No node is registered to handle the requested command.", []string{"Command name misspelled", "Node not yet spawned"}},
		{ErrorCodeRouteNotConnected,   "The target node is registered but not currently connected.", nil},
		{ErrorCodeRouteNotAvailable,   "The target node is connected but unable to accept the message.", nil},
		{ErrorCodeRouteNotAllowed,     "The caller does not have permission to use this route.", nil},
		{ErrorCodeRouteNotImplemented, "The route exists in the manifest but is not implemented.", nil},
		// Message
		{ErrorCodeMessageNotValid,  "The message is structurally valid but semantically invalid.", nil},
		{ErrorCodeMessageMalformed, "The message could not be parsed — it is syntactically malformed.", []string{"Missing handle prefix", "Unquoted value containing spaces"}},
		// Arguments
		{ErrorCodeArgumentMissing,      "A required argument was not provided.", []string{"SPORE.node.help called without node="}},
		{ErrorCodeArgumentInvalidType,  "An argument value does not match the expected type.", nil},
		{ErrorCodeArgumentConflict,     "Two or more arguments cannot be used together.", nil},
		{ErrorCodeArgumentOutOfRange,   "An argument value is outside the allowed range.", nil},
		{ErrorCodeArgumentUnrecognized, "An argument name is not recognised by the handler.", nil},
		{ErrorCodeArgumentDuplicated,   "The same argument name was provided more than once.", nil},
		// Flags
		{ErrorCodeFlagConflict,     "Two or more flags cannot be used together.", nil},
		{ErrorCodeFlagUnrecognized, "A flag is not recognised by the handler.", nil},
		{ErrorCodeFlagDuplicated,   "The same flag was specified more than once.", nil},
		// Handles
		{ErrorCodeHandleMissing, "A handle (~token) is required but was not provided.", []string{"Cast sent without a ~handle"}},
		{ErrorCodeHandleInUse,   "The handle is already registered and awaiting a reply.", []string{"Duplicate handle used before the first reply arrived"}},
		{ErrorCodeHandleExpired, "The handle is no longer valid (reply window elapsed).", nil},
		// Reserved keywords
		{ErrorCodeReservedKeyword, "A reserved keyword was used where it is not allowed.", []string{"Manifest declares an input/output named 'cast' or 'capture'"}},
	}
}

// NewError creates the appropriate hub-generated error based on the type of the original message.
func NewError(original Message, code ErrorCode, what string) Message {
	switch {
	case original.IsCast():
		return &CastError{originalMessage: original, code: code, what: what, mid: -1}
	case original.IsCapture():
		return &CaptureError{originalMessage: original, code: code, what: what, mid: -1}
	case original.IsSpore():
		return &SporeError{originalMessage: original, code: code, what: what, mid: -1}
	default:
		return &CastError{originalMessage: original, code: code, what: what, mid: -1}
	}
}

// ---- CastError --------------------------------------------------------------
// Generated by the hub when it fails to route a cast to its receiver.

type CastError struct {
	originalMessage Message
	code            ErrorCode
	what            string
	mid             int64
	destination     string
}



func (e *CastError) IsCast() bool         { return true }
func (e *CastError) IsCapture() bool      { return false }
func (e *CastError) IsSpore() bool        { return false }
func (e *CastError) IsError() bool        { return true }
func (e *CastError) IsCustomError() bool  { return false }
func (e *CastError) IsCancelled() bool    { return false }
func (e *CastError) Parse(_ string, _ string) error { return nil }
func (e *CastError) Cast() string    { return e.originalMessage.Cast() }
func (e *CastError) Capture() string { return "SPORE.hub" }
func (e *CastError) Command() string { return e.originalMessage.Command() }
func (e *CastError) Handle() string  { return e.originalMessage.Handle() }
func (e *CastError) Code() ErrorCode { return e.code }
func (e *CastError) What() string    { return e.what }
func (e *CastError) ToString() string {
	return fmt.Sprintf(`~%s:%s error cast_error code=%s what="%s" capture=SPORE.hub`, e.Handle(), e.Command(), e.code, e.what)
}
func (e *CastError) ToJsonString() string   { return errorJson(e.Handle(), e.Command(), e.code, e.what, e.Capture()) }
func (e *CastError) SetMessageId(mid int64) { e.mid = mid }
func (e *CastError) MessageId() int64       { return e.mid }
func (e *CastError) Source() string         { return e.Cast() }
func (e *CastError) SetDestination(dst string) { e.destination = dst }
func (e *CastError) Destination() string    { return e.destination }

// ---- CaptureError -----------------------------------------------------------
// Generated by the hub when it fails to route a capture back to the caster.

type CaptureError struct {
	originalMessage Message
	code            ErrorCode
	what            string
	mid             int64
	destination     string
}



func (e *CaptureError) IsCast() bool         { return false }
func (e *CaptureError) IsCapture() bool      { return true }
func (e *CaptureError) IsSpore() bool        { return false }
func (e *CaptureError) IsError() bool        { return true }
func (e *CaptureError) IsCustomError() bool  { return false }
func (e *CaptureError) IsCancelled() bool    { return false }
func (e *CaptureError) Parse(_ string, _ string) error { return nil }
func (e *CaptureError) Cast() string    { return e.originalMessage.Cast() }
func (e *CaptureError) Capture() string { return "SPORE.hub" }
func (e *CaptureError) Command() string { return e.originalMessage.Command() }
func (e *CaptureError) Handle() string  { return e.originalMessage.Handle() }
func (e *CaptureError) Code() ErrorCode { return e.code }
func (e *CaptureError) What() string    { return e.what }
func (e *CaptureError) ToString() string {
	return fmt.Sprintf(`~%s:%s error capture_error code=%s what="%s" capture=SPORE.hub`, e.Handle(), e.Command(), e.code, e.what)
}
func (e *CaptureError) ToJsonString() string   { return errorJson(e.Handle(), e.Command(), e.code, e.what, e.Capture()) }
func (e *CaptureError) SetMessageId(mid int64) { e.mid = mid }
func (e *CaptureError) MessageId() int64       { return e.mid }
func (e *CaptureError) Source() string         { return e.originalMessage.Source() }
func (e *CaptureError) SetDestination(dst string) { e.destination = dst }
func (e *CaptureError) Destination() string    { return e.destination }

// ---- SporeError -------------------------------------------------------------
// Generated by the hub when a SPORE.* command handler returns an error.

type SporeError struct {
	originalMessage Message
	code            ErrorCode
	what            string
	mid             int64
	destination     string
}

func (e *SporeError) IsCast() bool         { return false }
func (e *SporeError) IsCapture() bool      { return false }
func (e *SporeError) IsSpore() bool        { return true }
func (e *SporeError) IsError() bool        { return true }
func (e *SporeError) IsCustomError() bool  { return false }
func (e *SporeError) IsCancelled() bool    { return false }
func (e *SporeError) Parse(_ string, _ string) error { return nil }
func (e *SporeError) Cast() string    { return e.originalMessage.Cast() }
func (e *SporeError) Capture() string { return "SPORE.hub" }
func (e *SporeError) Command() string { return e.originalMessage.Command() }
func (e *SporeError) Handle() string  { return e.originalMessage.Handle() }
func (e *SporeError) Code() ErrorCode { return e.code }
func (e *SporeError) What() string    { return e.what }
func (e *SporeError) ToString() string {
	return fmt.Sprintf(`~%s:%s error spore_error code=%s what="%s" capture=SPORE.hub`, e.Handle(), e.Command(), e.code, e.what)
}
func (e *SporeError) ToJsonString() string   { return errorJson(e.Handle(), e.Command(), e.code, e.what, e.Capture()) }
func (e *SporeError) SetMessageId(mid int64) { e.mid = mid }
func (e *SporeError) MessageId() int64       { return e.mid }
func (e *SporeError) Source() string         { return e.Cast() }
func (e *SporeError) SetDestination(dst string) { e.destination = dst }
func (e *SporeError) Destination() string    { return e.destination }

// ---- NodeError --------------------------------------------------------------
// Parsed from a wire error response sent by a node (e.g. ~h1:cmd error code=... what=...).

type NodeError struct {
	raw        string
	handle     string
	command    string
	cast       string
	capture    string
	code       ErrorCode
	what       string
	isCustom   bool
	originFlag string
	mid         int64
	destination string
}

func (e *NodeError) IsCast() bool         { return false }
func (e *NodeError) IsCapture() bool      { return false }
func (e *NodeError) IsSpore() bool        { return false }
func (e *NodeError) IsError() bool        { return true }
func (e *NodeError) IsCustomError() bool  { return e.isCustom }
func (e *NodeError) IsCancelled() bool    { return false }

func (e *NodeError) Parse(raw string, from string) error {
	e.raw = raw
	e.cast = from
	e.mid = -1

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
		token := strings.TrimPrefix(first, "~")
		halves := strings.SplitN(token, ":", 2)
		if len(halves) >= 1 {
			e.handle = halves[0]
		}
		if len(halves) >= 2 {
			e.command = halves[1]
		}
	}

	for _, part := range parts[1:] {
		switch part {
		case "error":
			e.isCustom = false
		case "custom_error":
			e.isCustom = true
		case "spore_error", "node_error", "cast_error", "capture_error":
			e.originFlag = part
		}
		sub := strings.SplitN(part, "=", 2)
		if len(sub) != 2 {
			continue
		}
		switch sub[0] {
		case "code":
			e.code = ErrorCode(sub[1])
		case "capture":
			e.capture = sub[1]
		case "cast":
			e.cast = sub[1]
		case "what":
			e.what = unquoteValue(sub[1])
		}
	}

	if e.originFlag == "" {
		e.originFlag = "node_error"
	}

	// Block nodes from emitting the hub-only SporeFailure code.
	if !e.isCustom && e.code == ErrorCodeSporeFailure {
		e.code = ErrorCodeProtocolFailure
	}

	// Inject capture= so the caster sees which node produced the error.
	e.raw = raw + " capture=" + e.cast
	e.capture = e.cast
	return nil
}

func (e *NodeError) Cast() string    { return e.cast }
func (e *NodeError) Capture() string { return e.capture }
func (e *NodeError) Command() string { return e.command }
func (e *NodeError) Handle() string  { return e.handle }
func (e *NodeError) Code() ErrorCode { return e.code }
func (e *NodeError) What() string    { return e.what }
func (e *NodeError) ToString() string   { return e.raw }
func (e *NodeError) ToJsonString() string { return errorJson(e.handle, e.command, e.code, e.what, e.capture) }
func (e *NodeError) SetMessageId(mid int64)    { e.mid = mid }
func (e *NodeError) MessageId() int64          { return e.mid }
func (e *NodeError) Source() string            { return e.capture }
func (e *NodeError) SetDestination(dst string) { e.destination = dst }
func (e *NodeError) Destination() string       { return e.destination }

// ---- shared helper ----------------------------------------------------------

func errorJson(handle, command string, code ErrorCode, what, capture string) string {
	result := map[string]interface{}{
		"handle":  handle,
		"command": command,
		"code":    string(code),
		"what":    what,
		"capture": capture,
	}
	data, err := json.Marshal(result)
	if err != nil {
		return "{}"
	}
	return string(data)
}
