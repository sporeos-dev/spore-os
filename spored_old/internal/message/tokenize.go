// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package message

import (
	"errors"
	"strings"
)

// Tokenize splits a Spore wire message string into tokens, respecting quoted
// strings, arrays, objects, and inline call expressions. It replaces
// strings.Fields everywhere in the message package.
//
// Token boundaries are whitespace that appears outside any delimiter context:
//
//   - "..."  — double-quoted string; spaces allowed; supports backslash escapes:
//     \\, \", \n, \r, \t. A backslash followed by any other character is
//     preserved as-is (backslash + character).
//   - '...'  — single-quoted string; content is opaque, spaces allowed, no escapes.
//   - {...}  — object; content is opaque, spaces allowed, not recursive.
//   - [...]  — array;  content is opaque, spaces allowed, not recursive.
//   - (...)  — inline call; nested (...) are depth-tracked to support nesting.
//
// Each delimiter type is an independent context. Inside {...} only } closes it;
// inside [...] only ] closes it; inside (...) only matching ) closes it.
//
// Because single-quoted strings have no escapes, each quote style can contain
// the other literally: "it's fine" and 'say "hi"' both work. A double-quoted
// string may contain a literal double-quote via the \" escape.
//
// Delimiters are retained in the output tokens so callers can identify the kind
// of a value by its first character. Use unquoteValue to strip quote delimiters
// from argument values when needed.
//
// Returns an error if any delimiter is left unclosed.
func Tokenize(raw string) ([]string, error) {
	tokens := make([]string, 0, 8)
	buf := make([]byte, 0, 64)
	i := 0
	n := len(raw)

	flush := func() {
		if len(buf) > 0 {
			tokens = append(tokens, string(buf))
			buf = buf[:0]
		}
	}

	for i < n {
		ch := raw[i]

		switch {

		case ch == ' ' || ch == '\t':
			flush()
			i++

		case ch == '\'':
			// Single-quoted string: no escape processing; content is opaque.
			buf = append(buf, ch)
			i++
			for i < n && raw[i] != '\'' {
				buf = append(buf, raw[i])
				i++
			}
			if i >= n {
				return nil, errors.New("unterminated single-quoted string")
			}
			buf = append(buf, raw[i]) // closing '
			i++

		case ch == '"':
			// Double-quoted string: backslash escapes are active.
			// \\ \", \n, \r, \t are recognised; any other \X is kept as-is.
			// The raw escape sequences are stored in the token; unquoteValue
			// decodes them when the value is consumed.
			buf = append(buf, ch)
			i++
			for i < n {
				if raw[i] == '\\' && i+1 < n {
					// Consume the backslash and the next character together so
					// that \" never prematurely closes the string.
					buf = append(buf, raw[i], raw[i+1])
					i += 2
				} else if raw[i] == '"' {
					break
				} else {
					buf = append(buf, raw[i])
					i++
				}
			}
			if i >= n {
				return nil, errors.New("unterminated double-quoted string")
			}
			buf = append(buf, raw[i]) // closing "
			i++

		case ch == '{':
			buf = append(buf, ch)
			i++
			for i < n && raw[i] != '}' {
				buf = append(buf, raw[i])
				i++
			}
			if i >= n {
				return nil, errors.New("unterminated object {")
			}
			buf = append(buf, raw[i]) // }
			i++

		case ch == '[':
			buf = append(buf, ch)
			i++
			for i < n && raw[i] != ']' {
				buf = append(buf, raw[i])
				i++
			}
			if i >= n {
				return nil, errors.New("unterminated array [")
			}
			buf = append(buf, raw[i]) // ]
			i++

		case ch == '(':
			depth := 1
			buf = append(buf, ch)
			i++
			for i < n && depth > 0 {
				if raw[i] == '(' {
					depth++
				} else if raw[i] == ')' {
					depth--
				}
				buf = append(buf, raw[i])
				i++
			}
			if depth != 0 {
				return nil, errors.New("unterminated inline call (")
			}

		default:
			buf = append(buf, ch)
			i++
		}
	}

	flush()
	return tokens, nil
}

// unquoteValue strips surrounding "..." or '...' delimiters from a value string
// and, for double-quoted values, decodes backslash escape sequences.
// If the value is not surrounded by a matching quote pair, it is returned unchanged.
// Objects {...} and arrays [...] are not affected.
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

// unescapeDoubleQuoted decodes backslash escape sequences within the inner
// content of a double-quoted string (surrounding quotes already stripped).
// Recognised sequences: \\ → \, \" → ", \n → newline, \r → CR, \t → tab.
// Any other \X sequence is preserved as-is (backslash + character).
func unescapeDoubleQuoted(s string) string {
	if !strings.ContainsRune(s, '\\') {
		return s // fast path: no escapes present
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
				// Unknown escape: preserve backslash and character.
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
