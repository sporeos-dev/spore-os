package permission

import (
	"encoding/base64"
	"spored/internal/event"
	"testing"
)

func TestMakeToken(t *testing.T) {
	first := makeToken()
	second := makeToken()
	for _, token := range []string{first, second} {
		data, err := base64.RawURLEncoding.DecodeString(token)
		if err != nil || len(data) != 32 {
			t.Fatalf("token is not 32 bytes encoded as URL-safe base64: %q, %v", token, err)
		}
	}
	if first == second {
		t.Fatal("generated tokens are identical")
	}
}

func TestHasToken(t *testing.T) {
	event.Init()
	for _, tt := range []struct {
		name    string
		message string
		token   string
		want    bool
	}{
		{"exact", "example.run permission.token=valid ~h", "valid", true},
		{"different", "example.run permission.token=wrong ~h", "valid", false},
		{"other argument", "example.run other.permission.token=valid ~h", "valid", false},
		{"inside quoted value", `example.run note="permission.token=valid" ~h`, "valid", false},
		{"absent", "example.run ~h", "valid", false},
		{"empty expected token", "example.run ~h", "", false},
		{"malformed", "", "valid", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasToken(tt.message, tt.token); got != tt.want {
				t.Errorf("hasToken(%q, %q) = %t, want %t", tt.message, tt.token, got, tt.want)
			}
		})
	}
}
