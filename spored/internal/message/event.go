package message

import "strings"

// SporeEvent formats the body of a spore_event witness message per the spec (§9.3).
// The Witness.Spore() method wraps this body with the "witness" prefix,
// the spore_event flag, and spore_time= — so callers only need to describe
// the event itself.
//
// level should be one of "info", "warn", or "error".
// what is a human-readable description (will be quoted automatically).
// extras are optional pre-formatted key=value tokens (e.g. "node=com.example.clock").
//
// Example output (before Witness.Spore wraps it):
//   SPORE.hub.event level=warn what="Routing error" error="no route found"
func SporeEvent(level string, what string, extras ...string) string {
	parts := []string{
		"SPORE.hub.event",
		"level=" + level,
		`what="` + what + `"`,
	}
	parts = append(parts, extras...)
	return strings.Join(parts, " ")
}
