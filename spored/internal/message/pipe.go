package message

import (
	"spored/internal/iface"
	"strconv"
	"strings"
)

type pipe struct {
	id     string
	raw    string
	parts  []string
}

func IsPipe(raw string) bool {
	if !strings.Contains(raw, "|") {
		return false
	}
	return len(splitPipe(raw)) > 1
}

func Pipe(raw string, id string) (iface.Message, bool) {
	parts := splitPipe(raw)
	if len(parts) < 2 {
		return nil, false
	}

	p := &pipe{
		id:     id,
		raw:    raw,
		parts:  parts,
	}

	return p, true
}

//
//
// iface.Message
//

// returns the next raw view and increments the index
func (p *pipe) Capability() string {
	return "n/a"
}

func (p *pipe) Cast() string {
	return p.id
}

func (p *pipe) Capture() string {
	return "n/a"
}

func (p *pipe) Handle() string {
	return "n/a"
}

// subverted to handle special keys
// subverted to handle indexes
func (p *pipe) Arg(key string) (string, bool) {
	switch key {
	case "raw":
		return p.raw, true
	case "cast":
		return p.id, true
	}

	// try to get index from key
	index, err := strconv.Atoi(key)
	if err != nil {
		return "bad-key", false
	} else if index < 0 {
		return "low-index", false
	} else if index >= len(p.parts) {
		return "high-index", false
	}
	return p.parts[index], true
}

func (p *pipe) ArgIf(key string, ifnot string) string {
	return ifnot
}

func (p *pipe) Flag(flag string) bool {
	return false
}

func (p *pipe) Wire() string {
	return p.raw
}

func (p *pipe) Witness() string {
	return "n/a"
}

//
//
// private
// internal
// helpers
//

func splitPipe(raw string) []string {
	var parts []string
	var buf strings.Builder

	inSingle := false
	inDouble := false
	bracketDepth := 0
	braceDepth := 0
	escaped := false

	for i := 0; i < len(raw); i++ {
		ch := raw[i]

		if escaped {
			buf.WriteByte(ch)
			escaped = false
			continue
		}

		if ch == '\\' && inDouble {
			buf.WriteByte(ch)
			escaped = true
			continue
		}

		switch ch {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
			buf.WriteByte(ch)
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
			buf.WriteByte(ch)
		case '[':
			if !inSingle && !inDouble {
				bracketDepth++
			}
			buf.WriteByte(ch)
		case ']':
			if !inSingle && !inDouble && bracketDepth > 0 {
				bracketDepth--
			}
			buf.WriteByte(ch)
		case '{':
			if !inSingle && !inDouble {
				braceDepth++
			}
			buf.WriteByte(ch)
		case '}':
			if !inSingle && !inDouble && braceDepth > 0 {
				braceDepth--
			}
			buf.WriteByte(ch)
		case '|':
			if !inSingle && !inDouble && bracketDepth == 0 && braceDepth == 0 {
				part := strings.TrimSpace(buf.String())
				parts = append(parts, part)
				buf.Reset()
			} else {
				buf.WriteByte(ch)
			}
		default:
			buf.WriteByte(ch)
		}
	}

	if buf.Len() > 0 {
		part := strings.TrimSpace(buf.String())
		parts = append(parts, part)
	}

	return parts
}

