package message

import (
	"log/slog"
	"strings"
	"unicode"
	"unicode/utf8"
)

// HasFlag reports whether a single message or pipe stage contains a top-level
// flag. It identifies fields but does not validate the complete message.
func HasFlag(raw, flag string) bool {
	found := false
	scanFlag(raw, flag, func(_, _ int) bool {
		found = true
		return false
	})
	return found
}

// RemoveFlag removes all top-level occurrences of flag from a single message
// or pipe stage. If scanning encounters malformed syntax, it logs an error and
// returns raw unchanged. It does not validate the complete message.
func RemoveFlag(raw, flag string) string {
	var out strings.Builder
	last := 0
	found := false
	if !scanFlag(raw, flag, func(start, end int) bool {
		out.WriteString(raw[last:start])
		last = end
		found = true
		return true
	}) {
		slog.Error("malformed message while removing flag")
		return raw
	}
	if !found {
		return raw
	}
	out.WriteString(raw[last:])
	return out.String()
}

func scanFlag(raw, flag string, visit func(start, end int) bool) bool {
	if flag == "" || !strings.Contains(raw, flag) {
		return true
	}

	return scanFields(raw, func(start, end, _ int, bare bool) bool {
		field := raw[start:end]
		if bare && !strings.HasPrefix(field, "~") && field != "|" && field == flag {
			return visit(start, end)
		}
		return true
	})
}

func scanFields(raw string, visit func(start, end, valueStart int, bare bool) bool) bool {
	valid, _ := scanFieldRange(raw, 0, len(raw), true, visit)
	return valid
}

func scanFieldRange(raw string, from, limit int, hasHeader bool, visit func(start, end, valueStart int, bare bool) bool) (bool, bool) {
	fieldIndex := 0
	publish := false
	for i := from; i < limit; {
		r, size := utf8.DecodeRuneInString(raw[i:limit])
		if unicode.IsSpace(r) {
			i += size
			continue
		}

		start := i
		valueStart := -1
		blockHeaderStart, blockHeaderEnd := -1, -1
		var quote rune
		var delimiters []rune
		bare := true
		for i < limit {
			r, size = utf8.DecodeRuneInString(raw[i:limit])
			if quote != 0 {
				if quote == '"' && r == '\\' && i+size < limit {
					_, nextSize := utf8.DecodeRuneInString(raw[i+size : limit])
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
			if i == valueStart && strings.HasPrefix(raw[i:limit], "<<") {
				end := BlockEnd(raw, i+2)
				if end < 0 || end > limit {
					return false, false
				}
				lineEnd := strings.IndexByte(raw[i+2:end], '\n')
				if lineEnd < 0 {
					return false, false
				}
				blockHeaderStart = i + 2
				blockHeaderEnd = blockHeaderStart + lineEnd
				i = end
				continue
			}
			switch r {
			case '"', '\'':
				bare = false
				quote = r
			case '[':
				bare = false
				delimiters = append(delimiters, ']')
			case '{':
				bare = false
				delimiters = append(delimiters, '}')
			case ']', '}':
				if len(delimiters) == 0 || delimiters[len(delimiters)-1] != r {
					return false, false
				}
				delimiters = delimiters[:len(delimiters)-1]
			case '=':
				bare = false
				if len(delimiters) == 0 && valueStart < 0 {
					valueStart = i + size
				}
			}
			i += size
		}
		if quote != 0 || len(delimiters) != 0 {
			return false, false
		}

		field := raw[start:i]
		if hasHeader && fieldIndex == 0 {
			publish = field == "publish"
		} else if !(publish && fieldIndex == 1) && !visit(start, i, valueStart, bare) {
			return true, false
		}
		if blockHeaderStart >= 0 {
			valid, keepGoing := scanFieldRange(raw, blockHeaderStart, blockHeaderEnd, false, visit)
			if !valid || !keepGoing {
				return valid, keepGoing
			}
		}
		fieldIndex++
	}
	return true, true
}

// BlockEnd returns the index after a block's closing marker, or -1 if absent.
// start is the index immediately following the opening << marker.
func BlockEnd(raw string, start int) int {
	for start < len(raw) {
		offset := strings.Index(raw[start:], "\n>>")
		if offset < 0 {
			return -1
		}
		end := start + offset + len("\n>>")
		if end == len(raw) {
			return end
		}
		r, _ := utf8.DecodeRuneInString(raw[end:])
		if unicode.IsSpace(r) {
			return end
		}
		start = end
	}
	return -1
}
