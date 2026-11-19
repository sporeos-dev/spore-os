package message

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestHasFlag(t *testing.T) {
	const flag = "request.permission"
	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{"request", "file.read request.permission ~h", true},
		{"after handle", "file.read ~h request.permission", true},
		{"response", "~h:file.read ok request.permission", true},
		{"publish", "publish file.topic request.permission", true},
		{"witness", "witness request.permission", true},
		{"whitespace", "file.read\t~h\u2003request.permission\n", true},
		{"absent", "file.read ~h", false},
		{"command is not a flag", "request.permission ~h", false},
		{"topic is not a flag", "publish request.permission", false},
		{"handle is not a flag", "file.read ~request.permission", false},
		{"argument is not a flag", "file.read request.permission=true ~h", false},
		{"prefix", "file.read request.permission.extra ~h", false},
		{"suffix", "file.read other.request.permission ~h", false},
		{"double quoted", `file.read body="please request.permission now" ~h`, false},
		{"single quoted", `file.read body='please request.permission now' ~h`, false},
		{"escaped double quote", `file.read body="escaped \" request.permission still quoted" ~h`, false},
		{"single quote is literal", `file.read body='backslash\' request.permission ~h`, true},
		{"array", `file.read items=[a, request.permission, b] ~h`, false},
		{"object", `file.read body={a: request.permission, b: 2} ~h`, false},
		{"nested values", `file.read body=[{key: "x", nested: [request.permission]}] ~h`, false},
		{"flag after nested values", `file.read body=[{key: "x]"}] request.permission ~h`, true},
		{"quoted bare token", `file.read "request.permission" ~h`, false},
		{"block", "file.read body=<<\nrequest.permission\n>> ~h", false},
		{"flag on block opening line", "file.read body=<< request.permission ~h\nbody\n>>", true},
		{"flag after block", "file.read body=<<\nrequest.permission\n>> request.permission ~h", true},
		{"multiple blocks", "file.read a=<<\nfirst\n>> b=<<\nrequest.permission\n>> ~h", false},
		{"block closing prefix is content", "file.read body=<<\n>>not-end request.permission\n>> ~h", false},
		{"unclosed block", "file.read body=<<\nrequest.permission", false},
		{"unclosed quote", `file.read body="request.permission ~h`, false},
		{"unclosed array", `file.read body=[request.permission ~h`, false},
		{"mismatched container", `file.read body=[{x: 1] request.permission ~h`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasFlag(tt.raw, flag); got != tt.want {
				t.Errorf("HasFlag(%q, %q) = %t, want %t", tt.raw, flag, got, tt.want)
			}
		})
	}
}

func TestHasFlagRejectsNonFlags(t *testing.T) {
	for _, flag := range []string{"", "~h", "key=value", `"quoted"`, "|"} {
		if HasFlag(`command ~h key=value "quoted" |`, flag) {
			t.Errorf("HasFlag accepted non-flag %q", flag)
		}
	}
}

func TestRemoveFlag(t *testing.T) {
	const flag = "request.permission"
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"absent", `file.read body="request.permission" ~h`, `file.read body="request.permission" ~h`},
		{"one flag", "file.read request.permission ~h", "file.read  ~h"},
		{"repeated flags", "file.read request.permission ~h request.permission", "file.read  ~h "},
		{"whitespace preserved", "file.read\trequest.permission\u2003~h", "file.read\t\u2003~h"},
		{"values and similar names", `file.read request.permission.extra body="request.permission" request.permission=true request.permission ~h`, `file.read request.permission.extra body="request.permission" request.permission=true  ~h`},
		{"nested values", `file.read body=[{key: "x]", flags: [request.permission]}] request.permission ~h`, `file.read body=[{key: "x]", flags: [request.permission]}]  ~h`},
		{"escaped quotes", `file.read body="quoted \" request.permission" request.permission ~h`, `file.read body="quoted \" request.permission"  ~h`},
		{"literal single quote", `file.read body='backslash\' request.permission ~h`, `file.read body='backslash\'  ~h`},
		{"block and closing line", "file.read body=<<\nrequest.permission\n>> request.permission ~h", "file.read body=<<\nrequest.permission\n>>  ~h"},
		{"block opening line", "file.read body=<< request.permission ~h\nbody\n>>", "file.read body=<<  ~h\nbody\n>>"},
		{"command is not a flag", "request.permission ~h", "request.permission ~h"},
		{"topic is not a flag", "publish request.permission", "publish request.permission"},
		{"handle is not a flag", "file.read ~request.permission", "file.read ~request.permission"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RemoveFlag(tt.raw, flag); got != tt.want {
				t.Errorf("RemoveFlag(%q, %q) = %q, want %q", tt.raw, flag, got, tt.want)
			}
		})
	}
}

func TestRemoveFlagMalformedMessage(t *testing.T) {
	var log bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&log, nil)))
	defer slog.SetDefault(previous)

	raw := `file.read request.permission body="unterminated`
	if got := RemoveFlag(raw, "request.permission"); got != raw {
		t.Errorf("RemoveFlag changed malformed message: %q", got)
	}
	if !strings.Contains(log.String(), "malformed message") {
		t.Errorf("missing malformed message log: %q", log.String())
	}
}
