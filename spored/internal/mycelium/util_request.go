package mycelium

import (
	"spored/internal/utilities/error"
	"spored/internal/witness"
	"strings"
	"unicode"
	"unicode/utf8"
)

func extractHandleFromRequest(raw string) (string, bool) {
	var delimiters []rune
	previousSpace := true

	for i := 0; i < len(raw); {
		ch, size := utf8.DecodeRuneInString(raw[i:])

		if len(delimiters) > 0 {
			top := delimiters[len(delimiters)-1]
			if top == '"' || top == '\'' {
				if top == '"' && ch == '\\' && i+size < len(raw) {
					_, nextSize := utf8.DecodeRuneInString(raw[i+size:])
					i += size + nextSize
					continue
				}
				if ch == top {
					delimiters = delimiters[:len(delimiters)-1]
				}
			} else {
				switch ch {
				case '"', '\'':
					delimiters = append(delimiters, ch)
				case '[':
					delimiters = append(delimiters, ']')
				case '{':
					delimiters = append(delimiters, '}')
				default:
					if ch == top {
						delimiters = delimiters[:len(delimiters)-1]
					}
				}
			}
			previousSpace = false
			i += size
			continue
		}

		switch ch {
		case '"', '\'':
			delimiters = append(delimiters, ch)
		case '[':
			delimiters = append(delimiters, ']')
		case '{':
			delimiters = append(delimiters, '}')
		case '~':
			if previousSpace {
				start := i + size
				end := start
				for end < len(raw) {
					next, nextSize := utf8.DecodeRuneInString(raw[end:])
					if next == ':' {
						witness.Error(error.Malformed, error.Mycelium, "request handle contains a colon")
						return "", false
					}
					if unicode.IsSpace(next) || next == '"' || next == '\'' || next == '[' || next == ']' || next == '{' || next == '}' {
						break
					}
					end += nextSize
				}
				if end > start {
					return raw[start:end], true
				}
			}
		}

		previousSpace = unicode.IsSpace(ch)
		i += size
	}

	witness.Error(error.Missing, error.Mycelium, "request handle not found")
	return "", false
}

func extractCommandFromRequest(raw string) (string, bool) {
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		witness.Error(error.Missing, error.Mycelium, "request command not found")
		return "", false
	}
	if strings.Contains(fields[0], ":") {
		witness.Error(error.Malformed, error.Mycelium, "request command contains a colon")
		return "", false
	}
	return fields[0], true
}
