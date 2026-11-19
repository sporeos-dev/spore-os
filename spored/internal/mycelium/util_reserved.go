package mycelium

import (
	"spored/internal/cparser"
	"spored/internal/utilities/error"
	"spored/internal/utilities/message"
	"spored/internal/utilities/out"
	"spored/internal/witness"
	"strings"
	"unicode"
	"unicode/utf8"
)

func ensureNoReservedArgument(raw, key string) bool {
	// Check both the shared parser and wire whitespace: its tokenizer only
	// splits on ASCII spaces, while other clients may recognize tabs and newlines.
	parsed, ok := cparser.Parse(raw)
	if !ok {
		return false
	}
	_, parsedArgument := parsed.Args[key]
	if parsedArgument || hasReservedArgument(raw, key) {
		witness.Error(error.ReservedLanguage, error.Mycelium, "hub-reserved argument supplied", out.Pair("argument", key))
		return false
	}
	return true
}

func hasReservedArgument(raw, reserved string) bool {
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
					_, nextSize := utf8.DecodeRuneInString(raw[i+size:])
					i += size + nextSize
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
					return false
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
			case '=':
				if len(delimiters) == 0 && valueStart < 0 {
					if raw[start:i] == reserved {
						return true
					}
					valueStart = i + size
				}
			}
			i += size
		}
	}
	return false
}
