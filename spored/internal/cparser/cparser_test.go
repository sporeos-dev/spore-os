package cparser

import "testing"

func TestParse_Request(t *testing.T) {
	pm, ok := Parse("clock.get_time timezone=UTC ~h1")
	if !ok {
		t.Fatal("expected ok")
	}
	if pm.Type != TypeRequest {
		t.Errorf("type: got %d, want %d", pm.Type, TypeRequest)
	}
	if pm.Command != "clock.get_time" {
		t.Errorf("command: got %q", pm.Command)
	}
	if pm.Handle != "h1" {
		t.Errorf("handle: got %q", pm.Handle)
	}
	if pm.Args["timezone"] != "UTC" {
		t.Errorf("arg timezone: got %q", pm.Args["timezone"])
	}
}

func TestParse_Request_QuotedArg(t *testing.T) {
	pm, ok := Parse(`logger.append entry="hello world" ~l1`)
	if !ok {
		t.Fatal("expected ok")
	}
	if pm.Type != TypeRequest {
		t.Errorf("type: got %d, want %d", pm.Type, TypeRequest)
	}
	// value must be unquoted
	if pm.Args["entry"] != "hello world" {
		t.Errorf("arg entry: got %q, want %q", pm.Args["entry"], "hello world")
	}
}

func TestParse_Request_MissingHandle(t *testing.T) {
	_, ok := Parse("clock.get_time timezone=UTC")
	if ok {
		t.Fatal("expected not ok for missing handle")
	}
}

func TestParse_Response_Ok(t *testing.T) {
	pm, ok := Parse("~h1:clock.get_time time=1234 ok cast=node.id")
	if !ok {
		t.Fatal("expected ok")
	}
	if pm.Type != TypeResponse {
		t.Errorf("type: got %d, want %d", pm.Type, TypeResponse)
	}
	if pm.Handle != "h1" {
		t.Errorf("handle: got %q", pm.Handle)
	}
	if pm.Command != "clock.get_time" {
		t.Errorf("command: got %q", pm.Command)
	}
	if pm.Args["time"] != "1234" {
		t.Errorf("arg time: got %q", pm.Args["time"])
	}
	found := false
	for _, f := range pm.Flags {
		if f == "ok" {
			found = true
		}
	}
	if !found {
		t.Error("expected ok flag")
	}
}

func TestParse_Response_MissingOkOrError(t *testing.T) {
	_, ok := Parse("~h1:clock.get_time time=1234 cast=node.id")
	if ok {
		t.Fatal("expected not ok — response lacks ok/error flag")
	}
}

func TestParse_Response_Error(t *testing.T) {
	pm, ok := Parse(`~h1:clock.get_time error code=NotFound what="no such resource"`)
	if !ok {
		t.Fatal("expected ok")
	}
	if pm.Type != TypeResponse {
		t.Errorf("type: got %d, want %d", pm.Type, TypeResponse)
	}
	if pm.Args["code"] != "NotFound" {
		t.Errorf("arg code: got %q", pm.Args["code"])
	}
	if pm.Args["what"] != "no such resource" {
		t.Errorf("arg what: got %q", pm.Args["what"])
	}
}

func TestParse_Publish(t *testing.T) {
	pm, ok := Parse("publish events.tick interval=1000")
	if !ok {
		t.Fatal("expected ok")
	}
	if pm.Type != TypePublish {
		t.Errorf("type: got %d, want %d", pm.Type, TypePublish)
	}
	if pm.Command != "events.tick" {
		t.Errorf("command: got %q", pm.Command)
	}
	if pm.Args["interval"] != "1000" {
		t.Errorf("arg interval: got %q", pm.Args["interval"])
	}
}

func TestParse_Empty(t *testing.T) {
	_, ok := Parse("")
	if ok {
		t.Fatal("expected not ok for empty input")
	}
}

func TestParse_Flags(t *testing.T) {
	pm, ok := Parse("fs.read path=/tmp recursive verbose ~r1")
	if !ok {
		t.Fatal("expected ok")
	}
	hasRecursive, hasVerbose := false, false
	for _, f := range pm.Flags {
		if f == "recursive" {
			hasRecursive = true
		}
		if f == "verbose" {
			hasVerbose = true
		}
	}
	if !hasRecursive {
		t.Error("expected recursive flag")
	}
	if !hasVerbose {
		t.Error("expected verbose flag")
	}
}
