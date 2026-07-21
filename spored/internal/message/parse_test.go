package message

import (
	"testing"
)

// --- tokenize ---

func TestTokenize_Simple(t *testing.T) {
	tokens, err := tokenize("clock.get_time timezone=UTC ~h1")
	if err != "" {
		t.Fatalf("unexpected error: %s", err)
	}
	want := []string{"clock.get_time", "timezone=UTC", "~h1"}
	if len(tokens) != len(want) {
		t.Fatalf("got %v, want %v", tokens, want)
	}
	for i := range want {
		if tokens[i] != want[i] {
			t.Errorf("token[%d]: got %q, want %q", i, tokens[i], want[i])
		}
	}
}

func TestTokenize_QuotedValue(t *testing.T) {
	tokens, err := tokenize(`logger.append entry="hello world" ~l1`)
	if err != "" {
		t.Fatalf("unexpected error: %s", err)
	}
	want := []string{"logger.append", `entry="hello world"`, "~l1"}
	for i := range want {
		if tokens[i] != want[i] {
			t.Errorf("token[%d]: got %q, want %q", i, tokens[i], want[i])
		}
	}
}

func TestTokenize_SingleQuotedValue(t *testing.T) {
	tokens, err := tokenize(`filesystem.read path='/tmp/my file' recursive ~r1`)
	if err != "" {
		t.Fatalf("unexpected error: %s", err)
	}
	want := []string{"filesystem.read", "path='/tmp/my file'", "recursive", "~r1"}
	for i := range want {
		if tokens[i] != want[i] {
			t.Errorf("token[%d]: got %q, want %q", i, tokens[i], want[i])
		}
	}
}

func TestTokenize_ArrayArg(t *testing.T) {
	tokens, err := tokenize("SPORE.node.list filters=[a, b, c] ~m1")
	if err != "" {
		t.Fatalf("unexpected error: %s", err)
	}
	want := []string{"SPORE.node.list", "filters=[a, b, c]", "~m1"}
	for i := range want {
		if tokens[i] != want[i] {
			t.Errorf("token[%d]: got %q, want %q", i, tokens[i], want[i])
		}
	}
}

func TestTokenize_InlineCall(t *testing.T) {
	tokens, err := tokenize("SPORE.node.install path=(dialog.file_picker filters=[*.yaml]) ~i1")
	if err != "" {
		t.Fatalf("unexpected error: %s", err)
	}
	want := []string{"SPORE.node.install", "path=(dialog.file_picker filters=[*.yaml])", "~i1"}
	for i := range want {
		if tokens[i] != want[i] {
			t.Errorf("token[%d]: got %q, want %q", i, tokens[i], want[i])
		}
	}
}

func TestTokenize_UnterminatedDouble(t *testing.T) {
	_, err := tokenize(`bad="unclosed`)
	if err == "" {
		t.Fatal("expected error for unterminated double-quoted string")
	}
}

func TestTokenize_UnterminatedSingle(t *testing.T) {
	_, err := tokenize(`bad='unclosed`)
	if err == "" {
		t.Fatal("expected error for unterminated single-quoted string")
	}
}

// --- unquoteValue ---

func TestUnquoteValue_Double(t *testing.T) {
	if got := unquoteValue(`"hello world"`); got != "hello world" {
		t.Errorf("got %q", got)
	}
}

func TestUnquoteValue_Single(t *testing.T) {
	if got := unquoteValue("'hello world'"); got != "hello world" {
		t.Errorf("got %q", got)
	}
}

func TestUnquoteValue_Plain(t *testing.T) {
	if got := unquoteValue("plain"); got != "plain" {
		t.Errorf("got %q", got)
	}
}

func TestUnquoteValue_Escapes(t *testing.T) {
	if got := unquoteValue(`"say \"hi\""`); got != `say "hi"` {
		t.Errorf("got %q", got)
	}
}

func TestUnquoteValue_NewlineEscape(t *testing.T) {
	if got := unquoteValue(`"line1\nline2"`); got != "line1\nline2" {
		t.Errorf("got %q", got)
	}
}

// --- parseRequest ---

func TestParseRequest_Full(t *testing.T) {
	p, err := parseRequest(`filesystem.read path=/tmp recursive verbose ~r1`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.command != "filesystem.read" {
		t.Errorf("command: got %q", p.command)
	}
	if p.args["path"] != "/tmp" {
		t.Errorf("args[path]: got %q", p.args["path"])
	}
	if len(p.flags) != 2 || p.flags[0] != "recursive" || p.flags[1] != "verbose" {
		t.Errorf("flags: got %v", p.flags)
	}
	if p.handle != "r1" {
		t.Errorf("handle: got %q", p.handle)
	}
}

func TestParseRequest_QuotedArg(t *testing.T) {
	p, err := parseRequest(`logger.append entry="hello world" ~l1`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.args["entry"] != "hello world" {
		t.Errorf("args[entry]: got %q", p.args["entry"])
	}
	if p.handle != "l1" {
		t.Errorf("handle: got %q", p.handle)
	}
}

func TestParseRequest_MissingHandle(t *testing.T) {
	_, err := parseRequest("clock.get_time timezone=UTC")
	if err == nil {
		t.Fatal("expected error for missing handle")
	}
}

func TestParseRequest_EmptyMessage(t *testing.T) {
	_, err := parseRequest("")
	if err == nil {
		t.Fatal("expected error for empty message")
	}
}

func TestParseRequest_MalformedToken(t *testing.T) {
	_, err := parseRequest(`bad="unclosed ~h1`)
	if err == nil {
		t.Fatal("expected error for malformed token")
	}
}

// --- parseResponse ---

func TestParseResponse_OkSuccess(t *testing.T) {
	p, err := parseResponse(`~h1:clock.get_time time="2026-03-13T14:32:00Z" ok capture=com.example.clock`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.handle != "h1" {
		t.Errorf("handle: got %q", p.handle)
	}
	if p.command != "clock.get_time" {
		t.Errorf("command: got %q", p.command)
	}
	if p.args["time"] != "2026-03-13T14:32:00Z" {
		t.Errorf("args[time]: got %q", p.args["time"])
	}
	if p.args["capture"] != "com.example.clock" {
		t.Errorf("args[capture]: got %q", p.args["capture"])
	}
	if !containsFlag(p.flags, "ok") {
		t.Errorf("expected ok flag, got %v", p.flags)
	}
}

func TestParseResponse_ErrorResponse(t *testing.T) {
	p, err := parseResponse(`~h1:clock.get_time error code=RouteNotConnected what="no node" capture=SPORE.hub`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.handle != "h1" {
		t.Errorf("handle: got %q", p.handle)
	}
	if !containsFlag(p.flags, "error") {
		t.Errorf("expected error flag, got %v", p.flags)
	}
	if p.args["code"] != "RouteNotConnected" {
		t.Errorf("args[code]: got %q", p.args["code"])
	}
}

func TestParseResponse_Cancelled(t *testing.T) {
	p, err := parseResponse(`~r1:dialog.file_picker cancelled capture=com.example.ui`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !containsFlag(p.flags, "cancelled") {
		t.Errorf("expected cancelled flag, got %v", p.flags)
	}
}

func TestParseResponse_MissingBindingPrefix(t *testing.T) {
	_, err := parseResponse(`clock.get_time time="2026-03-13" ok capture=com.example.clock`)
	if err == nil {
		t.Fatal("expected error: missing ~handle:subject prefix")
	}
}

func TestParseResponse_MissingColon(t *testing.T) {
	_, err := parseResponse(`~h1 time="2026-03-13" ok capture=com.example.clock`)
	if err == nil {
		t.Fatal("expected error: handle without :subject")
	}
}

func TestParseResponse_EmptyHandle(t *testing.T) {
	_, err := parseResponse(`~:clock.get_time ok capture=com.example.clock`)
	if err == nil {
		t.Fatal("expected error: empty handle")
	}
}

func TestParseResponse_EmptySubject(t *testing.T) {
	_, err := parseResponse(`~h1: ok capture=com.example.clock`)
	if err == nil {
		t.Fatal("expected error: empty subject")
	}
}

// --- parseBroadcast ---

func TestParseBroadcast_Full(t *testing.T) {
	p, err := parseBroadcast(`publish SPORE.node.spawned node=com.example.clock`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.command != "SPORE.node.spawned" {
		t.Errorf("command: got %q", p.command)
	}
	if p.args["node"] != "com.example.clock" {
		t.Errorf("args[node]: got %q", p.args["node"])
	}
	if p.handle != "" {
		t.Errorf("expected no handle, got %q", p.handle)
	}
}

func TestParseBroadcast_WithFlags(t *testing.T) {
	p, err := parseBroadcast(`publish alerts.fire level=warn critical`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.command != "alerts.fire" {
		t.Errorf("command: got %q", p.command)
	}
	if !containsFlag(p.flags, "critical") {
		t.Errorf("expected critical flag, got %v", p.flags)
	}
}

func TestParseBroadcast_MissingPublish(t *testing.T) {
	_, err := parseBroadcast(`SPORE.node.spawned node=com.example.clock`)
	if err == nil {
		t.Fatal("expected error: missing publish keyword")
	}
}

func TestParseBroadcast_MissingSubject(t *testing.T) {
	_, err := parseBroadcast(`publish`)
	if err == nil {
		t.Fatal("expected error: missing subject after publish")
	}
}

// --- Request constructor ---

func TestRequest_Command(t *testing.T) {
	m, err := Request(`clock.get_time timezone=UTC ~h1`, "com.example.caller")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Command() != "clock.get_time" {
		t.Errorf("Command(): got %q", m.Command())
	}
}

func TestRequest_Arg(t *testing.T) {
	m, err := Request(`clock.get_time timezone=UTC ~h1`, "com.example.caller")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	val, argErr := m.Arg("timezone")
	if argErr != nil {
		t.Fatalf("Arg error: %v", argErr)
	}
	if val != "UTC" {
		t.Errorf("Arg(timezone): got %q", val)
	}
}

func TestRequest_ArgMissing(t *testing.T) {
	m, err := Request(`clock.get_time ~h1`, "com.example.caller")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, argErr := m.Arg("timezone")
	if argErr == nil {
		t.Fatal("expected error for missing arg")
	}
}

func TestRequest_ArgIf(t *testing.T) {
	m, err := Request(`clock.get_time ~h1`, "com.example.caller")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := m.ArgIf("timezone", "Local"); got != "Local" {
		t.Errorf("ArgIf: got %q", got)
	}
}

func TestRequest_Flag(t *testing.T) {
	m, err := Request(`filesystem.read path=/tmp recursive ~r1`, "com.example.caller")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !m.Flag("recursive") {
		t.Error("expected Flag(recursive) == true")
	}
	if m.Flag("verbose") {
		t.Error("expected Flag(verbose) == false")
	}
}

func TestRequest_Handle(t *testing.T) {
	m, err := Request(`clock.get_time ~h1`, "com.example.caller")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Handle() != "h1" {
		t.Errorf("Handle(): got %q", m.Handle())
	}
}

func TestRequest_Cast(t *testing.T) {
	m, err := Request(`clock.get_time ~h1`, "com.example.caller")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Cast() != "com.example.caller" {
		t.Errorf("Cast(): got %q", m.Cast())
	}
}

func TestRequest_Get(t *testing.T) {
	raw := `clock.get_time timezone=UTC ~h1`
	m, err := Request(raw, "com.example.caller")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Get() != raw {
		t.Errorf("Get(): got %q", m.Get())
	}
}

func TestRequest_MissingHandle_Error(t *testing.T) {
	_, err := Request(`clock.get_time timezone=UTC`, "com.example.caller")
	if err == nil {
		t.Fatal("expected error for missing handle")
	}
}

// --- Response constructor ---

func TestResponse_OkFields(t *testing.T) {
	m, err := Response(`~h1:clock.get_time time="2026-03-13T14:32:00Z" ok capture=com.example.clock`, "com.example.clock")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Handle() != "h1" {
		t.Errorf("Handle(): got %q", m.Handle())
	}
	if m.Command() != "clock.get_time" {
		t.Errorf("Command(): got %q", m.Command())
	}
	if !m.Flag("ok") {
		t.Error("expected Flag(ok) == true")
	}
	if m.Flag("error") {
		t.Error("expected Flag(error) == false")
	}
	val, argErr := m.Arg("capture")
	if argErr != nil {
		t.Fatalf("Arg(capture): %v", argErr)
	}
	if val != "com.example.clock" {
		t.Errorf("Arg(capture): got %q", val)
	}
}

func TestResponse_ErrorFields(t *testing.T) {
	m, err := Response(`~h1:clock.get_time error code=RouteNotConnected what="no node" capture=SPORE.hub`, "SPORE.hub")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !m.Flag("error") {
		t.Error("expected Flag(error) == true")
	}
	code, _ := m.Arg("code")
	if code != "RouteNotConnected" {
		t.Errorf("Arg(code): got %q", code)
	}
}

func TestResponse_Malformed(t *testing.T) {
	_, err := Response(`clock.get_time ok`, "com.example.clock")
	if err == nil {
		t.Fatal("expected error: response without ~handle:subject")
	}
}

// --- Broadcast constructor ---

func TestBroadcast_Fields(t *testing.T) {
	m, err := Broadcast(`publish SPORE.node.spawned node=com.example.clock`, "SPORE.hub")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Command() != "SPORE.node.spawned" {
		t.Errorf("Command(): got %q", m.Command())
	}
	if m.Topic() != "SPORE.node.spawned" {
		t.Errorf("Topic(): got %q", m.Topic())
	}
	if m.Cast() != "SPORE.hub" {
		t.Errorf("Cast(): got %q", m.Cast())
	}
	val, argErr := m.Arg("node")
	if argErr != nil {
		t.Fatalf("Arg(node): %v", argErr)
	}
	if val != "com.example.clock" {
		t.Errorf("Arg(node): got %q", val)
	}
	if m.Handle() != "" {
		t.Errorf("Handle(): expected empty, got %q", m.Handle())
	}
}

func TestBroadcast_Malformed(t *testing.T) {
	_, err := Broadcast(`SPORE.node.spawned node=com.example.clock`, "SPORE.hub")
	if err == nil {
		t.Fatal("expected error: broadcast without publish prefix")
	}
}

// helper

func containsFlag(flags []string, target string) bool {
	for _, f := range flags {
		if f == target {
			return true
		}
	}
	return false
}
