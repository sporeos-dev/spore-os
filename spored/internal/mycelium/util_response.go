package mycelium

import (
	"spored/internal/utilities/error"
	"spored/internal/witness"
)

func extractHandleFromResponse(raw string) (string, bool) {
	if len(raw) < 3 || raw[0] != '~' {
		witness.Error(error.Missing, error.Mycelium, "response handle not found")
		return "", false
	}

	for i := 1; i < len(raw); i++ {
		if raw[i] == ':' {
			if i == 1 {
				witness.Error(error.Malformed, error.Mycelium, "response handle is empty")
				return "", false
			}
			return raw[1:i], true
		}
	}

	witness.Error(error.Malformed, error.Mycelium, "response handle missing colon")
	return "", false
}
