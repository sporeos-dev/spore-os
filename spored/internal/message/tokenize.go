// Copyright 2026 mharr
// SPDX-License-Identifier: AGPL-3.0-only

package message

import "errors"

// Tokenize splits a Spore wire message string into tokens, respecting quoted
// strings, arrays, objects, and inline call expressions. It replaces
// strings.Fields everywhere in the message package.
//
// Token boundaries are whitespace that appears outside any delimiter context:
//
//   - "..."  — double-quoted string; content is opaque, spaces allowed, no escapes.
//   - '...'  — single-quoted string; content is opaque, spaces allowed, no escapes.
//   - {...}  — object; content is opaque, spaces allowed, not recursive.
//   - [...]  — array;  content is opaque, spaces allowed, not recursive.
//   - (...)  — inline call; nested (...) are depth-tracked to support nesting.
//
// Each delimiter type is an independent context. Inside {...} only } closes it;
// inside [...] only ] closes it; inside (...) only matching ) closes it.
//
// Because both quote styles are supported, each can contain the other without
// any escaping: "it's fine" and 'say "hi"' both work. A string containing
// both quote characters in the same value cannot be expressed — this is a known
// v1d0 limitation.
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

		case ch == '"' || ch == '\'':
			close := ch
			buf = append(buf, ch)
			i++
			for i < n && raw[i] != close {
				buf = append(buf, raw[i])
				i++
			}
			if i >= n {
				if close == '"' {
					return nil, errors.New("unterminated double-quoted string")
				}
				return nil, errors.New("unterminated single-quoted string")
			}
			buf = append(buf, raw[i]) // closing quote
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

// unquoteValue strips surrounding "..." or '...' delimiters from a value string.
// If the value is not surrounded by a matching quote pair, it is returned unchanged.
// Objects {...} and arrays [...] are not affected.
func unquoteValue(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
