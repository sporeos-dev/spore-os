package mycelium

import (
	"crypto/rand"
	"encoding/hex"
	"slices"
	"spored/internal/cparser"
	"spored/internal/event"
	message_event "spored/internal/event/message"
	sync_event "spored/internal/event/sync"
	"spored/internal/utilities/error"
	"spored/internal/utilities/message"
	"spored/internal/witness"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

func (m *Mycelium) startPipe(raw, origin string) {
	stages := splitPipe(raw)
	if len(stages) < 2 {
		witness.Error(error.Malformed, error.Pipe, "invalid pipe")
		return
	}

	handle, ok := extractHandleFromRequest(stages[0])
	if !ok {
		return
	}
	command, ok := extractCommandFromRequest(stages[0])
	if !ok {
		command = "n/a"
	}
	if !m.handles.mapHandle(handle, origin) {
		response := message_event.ResponseError(error.HandleInUse, error.Pipe, "handle in use", handle, command, origin)
		event.Dispatch().Response.Notify(response)
		return
	}

	go m.runPipe(stages, origin, handle, command)
}

func (m *Mycelium) runPipe(stages []string, origin, handle, firstCommand string) {
	owned := true
	defer func() {
		if owned {
			m.handles.deleteHandle(handle)
		}
	}()
	send := func(response message_event.IResponseContext) {
		m.handles.deleteHandle(handle)
		owned = false
		event.Dispatch().Response.Notify(response)
	}
	fail := func(code error.Code, what string) {
		witness.Error(code, error.Pipe, what)
		send(message_event.ResponseError(code, error.Pipe, what, handle, firstCommand, origin))
	}

	var previous message_event.IResponseContext
	for i, raw := range stages {
		stage, ok := cparser.Parse(raw)
		if !ok || stage.Type != cparser.TypeRequest || stage.Command == "" {
			fail(error.Malformed, "invalid pipe stage")
			return
		}
		if (i == 0 && stage.Handle != handle) || (i > 0 && stage.Handle != "") {
			fail(error.Malformed, "only the first pipe stage may have a handle")
			return
		}
		if !ensureNoCast(raw) {
			fail(error.ReservedLanguage, "cast is hub-reserved")
			return
		}
		timeout, ok := pipeTimeout(stage.Args)
		if !ok {
			fail(error.Malformed, "invalid pipe timeout")
			return
		}

		fields, ok := pipeFields(raw)
		if !ok || len(fields) == 0 {
			fail(error.Malformed, "invalid pipe stage fields")
			return
		}
		requestFields := []string{fields[0]}
		for _, field := range fields[1:] {
			if strings.HasPrefix(field, "~") {
				continue
			}
			name, argument, ok := pipeFieldName(field)
			if !ok {
				fail(error.Malformed, "invalid pipe stage field")
				return
			}
			if argument && name == "timeout" {
				continue
			}
			requestFields = append(requestFields, field)
		}

		if previous != nil {
			inputs, ok := m.commands.retrieveInputs(stage.Command)
			if !ok {
				fail(error.Missing, "pipe stage command not found")
				return
			}
			forwarded, code, what := pipeForward(previous.Message(), stage, inputs)
			if code != "" {
				fail(code, what)
				return
			}
			requestFields = append(requestFields, forwarded...)
		}

		var random [16]byte
		if _, err := rand.Read(random[:]); err != nil {
			fail(error.Generic, "cannot generate pipe handle")
			return
		}
		stageHandle := "spore_pipe_" + hex.EncodeToString(random[:])
		requestFields = append(requestFields, "~"+stageHandle)
		request := strings.Join(requestFields, " ")
		if parsed, ok := cparser.ParseAndValidate(request); !ok || parsed.Type != cparser.TypeRequest {
			fail(error.Malformed, "invalid resolved pipe stage")
			return
		}

		cast := origin
		if i < len(stages)-1 {
			cast = "SPORE.pipe:" + origin
		}
		syncRequest := sync_event.RequestWithTimeout(request, cast, timeout)
		event.Dispatch().SynchronizedRequest.Notify(syncRequest)
		response := syncRequest.Response()
		if response == nil {
			fail(error.Generic, "pipe stage did not return a response")
			return
		}
		parsed, ok := cparser.ParseAndValidate(response.Message())
		if !ok || parsed.Type != cparser.TypeResponse || parsed.Handle != stageHandle {
			fail(error.Malformed, "invalid pipe stage response")
			return
		}
		if slices.Contains(parsed.Flags, "error") || slices.Contains(parsed.Flags, "cancelled") || i == len(stages)-1 {
			colon := strings.IndexByte(response.Message(), ':')
			if colon < 0 {
				fail(error.Malformed, "invalid pipe stage response")
				return
			}
			reply := "~" + handle + response.Message()[colon:]
			send(message_event.Response(reply, response.Origin(), handle, origin))
			return
		}
		previous = response
	}
}

func splitPipe(raw string) []string {
	var parts []string
	var delimiters []rune
	var quote rune
	start := 0
	for i := 0; i < len(raw); {
		r, size := utf8.DecodeRuneInString(raw[i:])
		if quote != 0 {
			if quote == '"' && r == '\\' && i+size < len(raw) {
				_, next := utf8.DecodeRuneInString(raw[i+size:])
				i += size + next
				continue
			}
			if r == quote {
				quote = 0
			}
			i += size
			continue
		}
		if len(delimiters) == 0 && r == '<' && i > 0 && raw[i-1] == '=' && strings.HasPrefix(raw[i:], "<<") {
			end := message.BlockEnd(raw, i+2)
			if end < 0 {
				break
			}
			i = end
			continue
		}
		switch r {
		case '"', '\'':
			quote = r
		case '[':
			delimiters = append(delimiters, ']')
		case '{':
			delimiters = append(delimiters, '}')
		case ']', '}':
			if len(delimiters) > 0 && delimiters[len(delimiters)-1] == r {
				delimiters = delimiters[:len(delimiters)-1]
			}
		case '|':
			if len(delimiters) == 0 {
				parts = append(parts, strings.TrimSpace(raw[start:i]))
				start = i + size
			}
		}
		i += size
	}
	return append(parts, strings.TrimSpace(raw[start:]))
}

func pipeTimeout(args map[string]string) (*time.Duration, bool) {
	raw, ok := args["timeout"]
	if !ok {
		return nil, true
	}
	if raw == "false" {
		timeout := time.Duration(0)
		return &timeout, true
	}
	seconds, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || seconds < 0 || seconds > int64((1<<63-1)/time.Second) {
		return nil, false
	}
	timeout := time.Duration(seconds) * time.Second
	if timeout == 0 {
		timeout = time.Nanosecond
	}
	return &timeout, true
}

func pipeForward(raw string, stage cparser.ParsedMessage, inputs map[string]struct{}) ([]string, error.Code, string) {
	response, ok := cparser.ParseAndValidate(raw)
	if !ok || response.Type != cparser.TypeResponse {
		return nil, error.Malformed, "invalid pipe response"
	}
	fields, ok := pipeFields(raw)
	if !ok || len(fields) == 0 {
		return nil, error.Malformed, "invalid pipe response fields"
	}
	var forwarded []string
	for _, field := range fields[1:] {
		name, argument, ok := pipeFieldName(field)
		if !ok {
			return nil, error.Malformed, "invalid pipe response field"
		}
		if pipeReserved(name) {
			continue
		}
		if _, ok := inputs[name]; !ok {
			continue
		}
		if _, exists := stage.Args[name]; exists || slices.Contains(stage.Flags, name) {
			return nil, error.Collision, "pipe field duplicates an explicit input"
		}
		if argument {
			if _, ok := response.Args[name]; !ok {
				return nil, error.Malformed, "invalid pipe response argument"
			}
		} else if !slices.Contains(response.Flags, name) {
			return nil, error.Malformed, "invalid pipe response flag"
		}
		forwarded = append(forwarded, field)
	}
	return forwarded, "", ""
}

func pipeReserved(name string) bool {
	switch name {
	case "cast", "capture", "code", "what", "timeout", "ok", "error", "cancelled":
		return true
	default:
		return strings.HasPrefix(name, "spore.")
	}
}

func pipeFieldName(field string) (string, bool, bool) {
	if !strings.Contains(field, "=") {
		return field, false, field != ""
	}
	parsed, ok := cparser.Parse("pipe.field " + field + " ~pipe_field")
	if !ok || len(parsed.Args) != 1 || len(parsed.Flags) != 0 {
		return "", false, false
	}
	for name := range parsed.Args {
		return name, true, true
	}
	return "", false, false
}

func pipeFields(raw string) ([]string, bool) {
	var fields []string
	for i := 0; i < len(raw); {
		r, size := utf8.DecodeRuneInString(raw[i:])
		if unicode.IsSpace(r) {
			i += size
			continue
		}
		start := i
		valueStart := -1
		var quote rune
		var delimiters []rune
		for i < len(raw) {
			r, size = utf8.DecodeRuneInString(raw[i:])
			if quote != 0 {
				if quote == '"' && r == '\\' && i+size < len(raw) {
					_, next := utf8.DecodeRuneInString(raw[i+size:])
					i += size + next
					continue
				}
				if r == quote {
					quote = 0
				}
				i += size
				continue
			}
			if len(delimiters) == 0 && unicode.IsSpace(r) {
				break
			}
			if i == valueStart && strings.HasPrefix(raw[i:], "<<") {
				end := message.BlockEnd(raw, i+2)
				if end < 0 {
					return nil, false
				}
				i = end
				continue
			}
			switch r {
			case '"', '\'':
				quote = r
			case '[':
				delimiters = append(delimiters, ']')
			case '{':
				delimiters = append(delimiters, '}')
			case ']', '}':
				if len(delimiters) == 0 || delimiters[len(delimiters)-1] != r {
					return nil, false
				}
				delimiters = delimiters[:len(delimiters)-1]
			case '=':
				if len(delimiters) == 0 && valueStart < 0 {
					valueStart = i + size
				}
			}
			i += size
		}
		if quote != 0 || len(delimiters) != 0 {
			return nil, false
		}
		fields = append(fields, raw[start:i])
	}
	return fields, true
}
