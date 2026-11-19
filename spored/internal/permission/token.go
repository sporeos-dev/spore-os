package permission

import (
	"crypto/rand"
	"encoding/base64"
	"spored/internal/utilities/message"
)

func makeToken() string {
	var data [32]byte
	rand.Read(data[:])
	return base64.RawURLEncoding.EncodeToString(data[:])
}

func hasToken(raw, token string) bool {
    value, found := message.GetArg(raw, "permission.token")
    return found && token != "" && value == token
}
