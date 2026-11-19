package message

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestHasArg(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		arg  string
		want bool
	}{
		{"unquoted", "file.read permission.token=valid ~h", "permission.token", true},
		{"empty value", `file.read permission.token="" ~h`, "permission.token", true},
		{"quoted key", `file.read "permission.token"=valid ~h`, "permission.token", true},
		{"mixed quoted key", `file.read permi"ssion".token=valid ~h`, "permission.token", true},
		{"unicode whitespace", "file.read\tpath=value\u2003~h", "path", true},
		{"response", "~h:file.read path=value ok", "path", true},
		{"publish", "publish file.topic path=value", "path", true},
		{"after block", "file.read body=<<\npath=hidden\n>> path=shown ~h", "path", true},
		{"on block opening line", "file.read body=<< path=shown ~h\npath=hidden\n>>", "path", true},
		{"bare flag", "file.read permission.token ~h", "permission.token", false},
		{"other argument", "file.read other.permission.token=valid ~h", "permission.token", false},
		{"inside quoted value", `file.read note="permission.token=valid" ~h`, "permission.token", false},
		{"inside nested value", `file.read data=[{key: "permission.token=valid"}] ~h`, "permission.token", false},
		{"inside block", "file.read body=<<\npermission.token=valid\n>> ~h", "permission.token", false},
		{"command is not an argument", "path=value ~h", "path", false},
		{"different name", "file.read path.extra=value ~h", "path", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasArg(tt.raw, tt.arg); got != tt.want {
				t.Errorf("HasArg(%q, %q) = %t, want %t", tt.raw, tt.arg, got, tt.want)
			}
		})
	}
}

func TestGetArg(t *testing.T) {
	tests := []struct {
		name  string
		raw   string
		arg   string
		value string
		found bool
	}{
		{"unquoted", "file.read permission.token=valid ~h", "permission.token", "valid", true},
		{"double quoted", `file.read path="a \"quote\" \\path\nline\rreturn\ttab\q" ~h`, "path", "a \"quote\" \\path\nline\rreturn\ttab\\q", true},
		{"single quoted", `file.read path='a\n"quoted"\path' ~h`, "path", `a\n"quoted"\path`, true},
		{"empty quoted value", `file.read path="" ~h`, "path", "", true},
		{"array", `file.read items=[a, {nested: "x y"}] ~h`, "items", `[a, {nested: "x y"}]`, true},
		{"object", `file.read body={key: "x y"} ~h`, "body", `{key: "x y"}`, true},
		{"block", "file.read body=<<\nline one\nline two\n>> ~h", "body", "line one\nline two", true},
		{"block preserves raw carriage return", "file.read body=<<\nline\r\n>> ~h", "body", "line\r", true},
		{"block opening line", "file.read body=<< ~h path=visible\nline one\nline two\n>>", "body", "line one\nline two", true},
		{"argument on block opening line", "file.read body=<< ~h path=visible\npath=hidden\n>>", "path", "visible", true},
		{"after block", "file.read body=<<\npath=hidden\n>> path=shown ~h", "path", "shown", true},
		{"repeated argument", "file.read path=first path=last ~h", "path", "last", true},
		{"quoted key", `file.read "path"=value ~h`, "path", "value", true},
		{"mixed quoted key and value", `file.read pa"th"=a"b c"d ~h`, "path", "ab cd", true},
		{"adjacent quoted sections", `file.read path="one"'two' ~h`, "path", "onetwo", true},
		{"nested quotes preserved", `file.read path=prefix[{"key": "value"}]" end" ~h`, "path", `prefix[{"key": "value"}] end`, true},
		{"unicode whitespace", "file.read\tpath=value\u2003~h", "path", "value", true},
		{"response", "~h:file.read path=value ok", "path", "value", true},
		{"publish", "publish file.topic path=value", "path", "value", true},
		{"absent", "file.read other=value ~h", "path", "", false},
		{"command is not an argument", "path=value ~h", "path", "", false},
		{"different argument", "file.read other.permission.token=valid ~h", "permission.token", "", false},
		{"inside quoted value", `file.read note="permission.token=valid" ~h`, "permission.token", "", false},
		{"inside array", "file.read items=[permission.token=valid] ~h", "permission.token", "", false},
		{"inside block", "file.read body=<<\npermission.token=valid\n>> ~h", "permission.token", "", false},
		{"bare flag", "file.read permission.token ~h", "permission.token", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, found := GetArg(tt.raw, tt.arg)
			if value != tt.value || found != tt.found {
				t.Errorf("GetArg(%q, %q) = %q, %t; want %q, %t",
					tt.raw, tt.arg, value, found, tt.value, tt.found)
			}
		})
	}
}

func TestGetArgIf(t *testing.T) {
	if got := GetArgIf(`file.read path="" ~h`, "path", "fallback"); got != "" {
		t.Errorf("empty argument = %q, want empty value", got)
	}
	if got := GetArgIf("file.read ~h", "path", "fallback"); got != "fallback" {
		t.Errorf("missing argument = %q, want fallback", got)
	}
}

func TestGetArgMalformedMessage(t *testing.T) {
	var log bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&log, nil)))
	defer slog.SetDefault(previous)

	value, found := GetArg(`file.read path=value body="unterminated`, "path")
	if found || value != "" {
		t.Errorf("GetArg returned %q, %t for malformed message", value, found)
	}
	if !strings.Contains(log.String(), "malformed message") {
		t.Errorf("missing malformed message log: %q", log.String())
	}
}
