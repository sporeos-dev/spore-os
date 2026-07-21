package message

import (
	"strings"

	"spored/internal/utilities/error"
)

// tokenize splits a raw Spore wire message into tokens, respecting quoted
// strings, arrays, objects, and inline call expressions. Token boundaries
// are whitespace that appears outside any delimiter context:
//
//   - "..."  — double-quoted string; supports backslash escapes: \\, \", \n, \r, \t
//   - '...'  — single-quoted string; content is opaque, no escapes
//   - {...}  — object; content is opaque
//   - [...]  — array; content is opaque
//   - (...)  — inline call; nested parens are depth-tracked
//
// Returns an error string (non-empty) if any delimiter is unclosed.
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
			buf = append(buf, raw[i]) // closing '
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
				return nil, "unterminated object {"
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
				return nil, "unterminated array ["
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

// unquoteValue strips surrounding quote delimiters from a value token and
// decodes escape sequences for double-quoted strings. Arrays ([...]) and
// objects ({...}) are returned unchanged.
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

// parsedMessage holds the structured result of parsing a Spore wire message.
type parsedMessage struct {
	command string
	args    map[string]string
	flags   []string
	handle  string
}

// classifyTokens classifies a slice of already-tokenised tokens into
// args (key=value), flags (bare tokens), and handle (~token).
// It is used by the message-type-specific parse functions below.
func classifyTokens(tokens []string) (args map[string]string, flags []string, handle string) {
	args = make(map[string]string)
	flags = []string{}
	for _, tok := range tokens {
		switch {
		case strings.HasPrefix(tok, "~"):
			handle = strings.TrimPrefix(tok, "~")
		case strings.ContainsRune(tok, '='):
			eq := strings.IndexByte(tok, '=')
			key := tok[:eq]
			val := unquoteValue(tok[eq+1:])
			args[key] = val
		default:
			flags = append(flags, tok)
		}
	}
	return
}

// parseRequest parses a request wire message:
//
//	subject [key=value ...] [flag ...] ~handle
//
// Returns Malformed if the raw string is empty or cannot be tokenized.
// Returns Missing if no ~handle token is present (every call must include one).
func parseRequest(raw string) (parsedMessage, *error.Error) {
	out := parsedMessage{args: make(map[string]string), flags: []string{}}

	if strings.TrimSpace(raw) == "" {
		return out, error.New(error.Malformed, error.Message, "empty message")
	}

	tokens, errStr := tokenize(raw)
	if errStr != "" {
		return out, error.New(error.Malformed, error.Message, errStr)
	}
	if len(tokens) == 0 {
		return out, error.New(error.Malformed, error.Message, "empty message")
	}

	out.command = tokens[0]
	out.args, out.flags, out.handle = classifyTokens(tokens[1:])

	if out.handle == "" {
		return out, error.New(error.Missing, error.Message, "handle missing: every call must include a ~handle")
	}

	return out, nil
}

// parseResponse parses a response wire message:
//
//	~handle:subject [key=value ...] ok|error|custom_error|cancelled [capture=node_id]
//
// The hub injects ok and capture= before delivering to a caller; a response
// received directly from a node may not yet carry them — both forms are valid.
//
// Returns Malformed if the raw string is empty, cannot be tokenized, or the
// first token is not in the required ~handle:subject form.
func parseResponse(raw string) (parsedMessage, *error.Error) {
	out := parsedMessage{args: make(map[string]string), flags: []string{}}

	if strings.TrimSpace(raw) == "" {
		return out, error.New(error.Malformed, error.Message, "empty message")
	}

	tokens, errStr := tokenize(raw)
	if errStr != "" {
		return out, error.New(error.Malformed, error.Message, errStr)
	}
	if len(tokens) == 0 {
		return out, error.New(error.Malformed, error.Message, "empty message")
	}

	// First token must be ~handle:subject
	first := tokens[0]
	if !strings.HasPrefix(first, "~") {
		return out, error.New(error.Malformed, error.Message, "response must begin with ~handle:subject")
	}

	binding := first[1:] // strip leading ~
	colon := strings.IndexByte(binding, ':')
	if colon < 0 {
		return out, error.New(error.Malformed, error.Message, "response first token must be in ~handle:subject form")
	}

	out.handle = binding[:colon]
	out.command = binding[colon+1:]

	if out.handle == "" {
		return out, error.New(error.Malformed, error.Message, "handle is empty in ~handle:subject")
	}
	if out.command == "" {
		return out, error.New(error.Malformed, error.Message, "subject is empty in ~handle:subject")
	}

	// Remaining tokens are args and flags; responses never carry another ~ token.
	out.args, out.flags, _ = classifyTokens(tokens[1:])

	return out, nil
}

// parseBroadcast parses a broadcast wire message:
//
//	publish subject [key=value ...] [flags]
//
// Returns Malformed if the message does not begin with the reserved publish
// keyword, or if no subject follows it.
func parseBroadcast(raw string) (parsedMessage, *error.Error) {
	out := parsedMessage{args: make(map[string]string), flags: []string{}}

	if strings.TrimSpace(raw) == "" {
		return out, error.New(error.Malformed, error.Message, "empty message")
	}

	tokens, errStr := tokenize(raw)
	if errStr != "" {
		return out, error.New(error.Malformed, error.Message, errStr)
	}

	if len(tokens) == 0 || tokens[0] != "publish" {
		return out, error.New(error.Malformed, error.Message, "broadcast must begin with the publish keyword")
	}
	if len(tokens) < 2 {
		return out, error.New(error.Malformed, error.Message, "broadcast missing subject after publish")
	}

	out.command = tokens[1] // topic/subject
	out.args, out.flags, _ = classifyTokens(tokens[2:])

	return out, nil
}
