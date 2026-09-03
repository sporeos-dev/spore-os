package cparser

/*
#cgo CFLAGS: -I${SRCDIR}/../../../../spore-client-libs/parser/include
#cgo darwin LDFLAGS: -L${SRCDIR}/../../../../spore-client-libs/dist -lspore_parser -lc++
#cgo linux  LDFLAGS: -L${SRCDIR}/../../../../spore-client-libs/dist -lspore_parser -lstdc++
#include <stddef.h>
#include <stdbool.h>
#include <stdlib.h>
#include "spore_parser.h"
*/
import "C"
import (
	"spored/internal/utilities/error"
	"spored/internal/witness"
	"strings"
	"unsafe"
)

// Message type constants mirror spore_parser_type_t values.
const (
	TypeUnknown  = int(C.SPORE_PARSER_TYPE_UNKNOWN)
	TypeRequest  = int(C.SPORE_PARSER_TYPE_REQUEST)
	TypeResponse = int(C.SPORE_PARSER_TYPE_RESPONSE)
	TypeWitness  = int(C.SPORE_PARSER_TYPE_WITNESS)
	TypePublish  = int(C.SPORE_PARSER_TYPE_PUBLISH)
)

// ParsedMessage holds the result of a successful parse.
type ParsedMessage struct {
	Type    int
	Command string            // capability name for requests, ~handle: subject for responses, topic for publish
	Handle  string            // request ~handle or response ~handle
	Args    map[string]string // arg values are unquoted
	Flags   []string
}

// Parse invokes the shared C parser library on raw and returns the structured
// message. Returns (zero value, false) when the parser reports an error.
func Parse(raw string) (ParsedMessage, bool) {
	out := ParsedMessage{
		Args:  make(map[string]string),
		Flags: []string{},
	}

	if strings.TrimSpace(raw) == "" {
		return out, false
	}

	// trace boolean init false
	t := C.bool(false)
	parser := C.spore_parser_create(t)
	if parser == nil {
		return out, false
	}
	defer C.spore_parser_destroy(parser)

	msg := C.spore_message_create()
	if msg == nil {
		return out, false
	}
	defer C.spore_message_destroy(msg)

	cRaw := C.CString(raw)
	defer C.free(unsafe.Pointer(cRaw))

	C.spore_parse(parser, cRaw, C.size_t(len(raw)), msg)

	if bool(C.spore_parser_has_error(parser)) {
		code := C.GoString(C.spore_parser_get_error_code(parser))
		what := C.GoString(C.spore_parser_get_error_what(parser))
		witness.Send(
			error.New(
				error.Code(code),
				error.Parser,
				what))
		return out, false
	}

	out.Type = int(C.spore_parser_get_type(parser))

	if cap := C.spore_message_get_capability(msg); cap != nil {
		out.Command = C.GoString(cap)
	}
	if h := C.spore_message_get_handle(msg); h != nil {
		out.Handle = C.GoString(h)
	}

	var numArgs C.size_t
	rawArgs := C.spore_message_get_args(msg, &numArgs)
	if rawArgs != nil && numArgs > 0 {
		for _, a := range unsafe.Slice(rawArgs, int(numArgs)) {
			key := C.GoString(a.pKey)
			val := unquoteValue(C.GoString(a.pValue))
			out.Args[key] = val
		}
	}

	var numFlags C.size_t
	rawFlags := C.spore_message_get_flags(msg, &numFlags)
	if rawFlags != nil && numFlags > 0 {
		for _, f := range unsafe.Slice(rawFlags, int(numFlags)) {
			out.Flags = append(out.Flags, C.GoString(f))
		}
	}

	return out, true
}

// unquoteValue strips surrounding delimiters and decodes escape sequences,
// mirroring what the Go tokeniser did before this layer existed.
func unquoteValue(s string) string {
	if len(s) >= 2 {
		if s[0] == '"' && s[len(s)-1] == '"' {
			return unescapeDoubleQuoted(s[1 : len(s)-1])
		}
		if s[0] == '\'' && s[len(s)-1] == '\'' {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func unescapeDoubleQuoted(s string) string {
	if !strings.ContainsRune(s, '\\') {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case '\\':
				b.WriteByte('\\')
			case '"':
				b.WriteByte('"')
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			default:
				b.WriteByte('\\')
				b.WriteByte(s[i+1])
			}
			i += 2
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}
