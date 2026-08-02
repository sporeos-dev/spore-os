package message

import (
	"strings"

	"spored/internal/utilities/error"
	"spored/internal/utilities/parse"
)

// tokenize wraps parse.Tokenize for use within the message package.
func tokenize(raw string) ([]string, string) {
	return parse.Tokenize(raw)
}

func unquoteValue(s string) string {
	return parse.UnquoteValue(s)
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
