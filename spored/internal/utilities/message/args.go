package message

import (
	"log/slog"
	"spored/internal/utilities/parse"
	"strings"
)

// HasArg reports whether a single message or pipe stage contains a top-level
// argument. It identifies fields but does not validate the complete message.
func HasArg(raw, arg string) bool {
	if strings.TrimSpace(raw) == "" || arg == "" {
		slog.Error("cannot check argument in empty message or with empty name")
		return false
	}

	found := false
	if !scanFields(raw, func(start, _ int, valueStart int, _ bool) bool {
		if valueStart >= 0 && decodeArgText(raw[start:valueStart-1]) == arg {
			found = true
			return false
		}
		return true
	}) {
		slog.Error("malformed message while checking argument")
		return false
	}
	return found
}

// GetArg returns the decoded value of a top-level argument in a single message
// or pipe stage. It identifies fields but does not validate the complete message.
func GetArg(raw, arg string) (string, bool) {
	if strings.TrimSpace(raw) == "" || arg == "" {
		slog.Error("cannot get argument from empty message or with empty name")
		return "", false
	}

	var value string
	found := false
	if !scanFields(raw, func(start, end, valueStart int, _ bool) bool {
		if valueStart < 0 || decodeArgText(raw[start:valueStart-1]) != arg {
			return true
		}

		value = raw[valueStart:end]
		if strings.HasPrefix(value, "<<") && strings.HasSuffix(value, ">>") {
			content := value[2 : len(value)-2]
			value = content[strings.IndexByte(content, '\n')+1:]
			value = strings.TrimSuffix(value, "\n")
		} else {
			value = decodeArgText(value)
		}
		found = true
		return true
	}) {
		slog.Error("malformed message while getting argument")
		return "", false
	}
	return value, found
}

// GetArgIf returns the argument value, or fallback if the argument is absent.
func GetArgIf(raw, arg, fallback string) string {
	value, ok := GetArg(raw, arg)
	if !ok {
		return fallback
	}
	return value
}

// scanFields checks quote and container balance before decoding a field.
func decodeArgText(raw string) string {
	var out strings.Builder
	last := 0
	var delimiters []byte
	for i := 0; i < len(raw); {
		switch raw[i] {
		case '"', '\'':
			end := quoteEnd(raw, i)
			if len(delimiters) == 0 {
				out.WriteString(raw[last:i])
				out.WriteString(parse.UnquoteValue(raw[i:end]))
				last = end
			}
			i = end
		case '[', '{':
			delimiters = append(delimiters, raw[i])
			i++
		case ']', '}':
			delimiters = delimiters[:len(delimiters)-1]
			i++
		default:
			i++
		}
	}
	if last == 0 {
		return raw
	}
	out.WriteString(raw[last:])
	return out.String()
}

func quoteEnd(raw string, start int) int {
	quote := raw[start]
	for i := start + 1; i < len(raw); i++ {
		if quote == '"' && raw[i] == '\\' {
			i++
		} else if raw[i] == quote {
			return i + 1
		}
	}
	return len(raw)
}
