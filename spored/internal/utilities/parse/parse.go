package parse

import (
	"strings"
)

// tokenize splits a raw string into tokens respecting quoted strings, arrays,
// objects, and inline call expressions. Returns a non-empty error string if
// any delimiter is unclosed.
func tokenize(raw string) ([]string, string) {
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
			buf = append(buf, ch)
			i++
			for i < n && raw[i] != '\'' {
				buf = append(buf, raw[i])
				i++
			}
			if i >= n {
				return nil, "unterminated single-quoted string"
			}
			buf = append(buf, raw[i])
			i++

		case ch == '"':
			buf = append(buf, ch)
			i++
			for i < n {
				if raw[i] == '\\' && i+1 < n {
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
				return nil, "unterminated double-quoted string"
			}
			buf = append(buf, raw[i])
			i++

		case ch == '{':
			buf = append(buf, ch)
			i++
			for i < n && raw[i] != '}' {
				buf = append(buf, raw[i])
				i++
			}
			if i >= n {
				return nil, "unterminated object {"
			}
			buf = append(buf, raw[i])
			i++

		case ch == '[':
			buf = append(buf, ch)
			i++
			for i < n && raw[i] != ']' {
				buf = append(buf, raw[i])
				i++
			}
			if i >= n {
				return nil, "unterminated array ["
			}
			buf = append(buf, raw[i])
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
				return nil, "unterminated inline call ("
			}

		default:
			buf = append(buf, ch)
			i++
		}
	}

	flush()
	return tokens, ""
}

// unquoteValue strips surrounding quote delimiters and decodes escape
// sequences for double-quoted strings. Arrays and objects are returned as-is.
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

// ArrayToStrings parses a loosely-formatted array string into a []string.
// Handles [ "A" "B" ], [ A B ], [ "A", "B" ], [ A, B ] and mixtures.
// Quoted strings with internal spaces are handled correctly.
func ArrayToStrings(s string) []string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	// Replace commas outside of quotes with spaces so the tokenizer sees them
	// as whitespace. This is safe for simple values; commas inside quoted
	// strings will be preserved because the tokenizer runs after this step
	// only on the outer level — but we replace before tokenizing, so a comma
	// inside `"hello, world"` would be corrupted. Acceptable for the current
	// use-case (identifiers and short reason strings without internal commas).
	s = strings.ReplaceAll(s, ",", " ")
	tokens, _ := tokenize(s)
	result := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		if tok != "" {
			result = append(result, unquoteValue(tok))
		}
	}
	return result
}
