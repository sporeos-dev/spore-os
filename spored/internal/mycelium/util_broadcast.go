package mycelium

import (
	"spored/internal/utilities/error"
	"spored/internal/witness"
	"strings"
)

func extractTopic(raw string) (string, bool) {
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		witness.Error(error.Missing, error.Mycelium, "broadcast topic not found")
		return "", false
	}
	return fields[0], true
}
