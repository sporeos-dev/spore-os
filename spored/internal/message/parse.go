package message

import "spored/internal/cparser"

// parsedMessage holds the structured result of parsing a Spore wire message.
type parsedMessage struct {
	command string
	args    map[string]string
	flags   []string
	handle  string
}

// parseRequest parses a request wire message:
//
//	subject [key=value ...] [flag ...] ~handle
func parseRequest(raw string) (parsedMessage, bool) {
	pm, ok := cparser.Parse(raw)
	if !ok || pm.Type != cparser.TypeRequest {
		return parsedMessage{}, false
	}
	return parsedMessage{
		command: pm.Command,
		args:    pm.Args,
		flags:   pm.Flags,
		handle:  pm.Handle,
	}, true
}

// parseResponse parses a response wire message:
//
//	~handle:subject [key=value ...] ok|error [capture=node_id]
//
// The C parser enforces that exactly one of ok or error is present, and that
// error responses carry code= and what= args.
func parseResponse(raw string) (parsedMessage, bool) {
	pm, ok := cparser.Parse(raw)
	if !ok || pm.Type != cparser.TypeResponse {
		return parsedMessage{}, false
	}
	return parsedMessage{
		command: pm.Command,
		args:    pm.Args,
		flags:   pm.Flags,
		handle:  pm.Handle,
	}, true
}

// parseBroadcast parses a publish wire message:
//
//	publish subject [key=value ...] [flags]
func parseBroadcast(raw string) (parsedMessage, bool) {
	pm, ok := cparser.Parse(raw)
	if !ok || pm.Type != cparser.TypePublish {
		return parsedMessage{}, false
	}
	return parsedMessage{
		command: pm.Command,
		args:    pm.Args,
		flags:   pm.Flags,
	}, true
}
