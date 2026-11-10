package mycelium

import (
	"spored/internal/base"
	"spored/internal/event"
	witness_event "spored/internal/event/witness"
	"strings"
	"testing"
)

type extractionWitnessSubscriber struct {
	*base.Base
	events []witness_event.IWitnessContext
}

func (s *extractionWitnessSubscriber) OnWitness(ctx witness_event.IWitnessContext) {
	s.events = append(s.events, ctx)
}

func TestExtractHandleFromRequest(t *testing.T) {
	event.Init()
	tests := []struct {
		name string
		raw  string
		want string
		ok   bool
	}{
		{"sentence", "this is ~handle a sentence", "handle", true},
		{"start", "~handle", "handle", true},
		{"hyphen", "~my-handle", "my-handle", true},
		{"whitespace", "text\t~handle\nnext", "handle", true},
		{"unicode whitespace", "text\u2003~handle", "handle", true},
		{"embedded tilde", "somethi~ng", "", false},
		{"punctuation before tilde", ",~name", "", false},
		{"empty", "text ~ ", "", false},
		{"empty before valid", "~ ~valid", "valid", true},
		{"double quoted", `text "~hidden" ~visible`, "visible", true},
		{"single quoted", `text '~hidden' ~visible`, "visible", true},
		{"array", `text [~hidden] ~visible`, "visible", true},
		{"object", `text {~hidden} ~visible`, "visible", true},
		{"nested delimiters", `text [{key="[~hidden]"} ~hidden] ~visible`, "visible", true},
		{"escaped double quote", `text "escaped \" ~hidden" ~visible`, "visible", true},
		{"unclosed delimiter", `text [ ~hidden`, "", false},
		{"colon", "~name:subject ~valid", "", false},
		{"colon in excluded text", `text "~name:subject" ~valid`, "valid", true},
		{"first handle", "~first ~second", "first", true},
		{"delimiter after handle", `~handle[~hidden]`, "handle", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extractHandleFromRequest(tt.raw)
			if got != tt.want || ok != tt.ok {
				t.Errorf("extractHandleFromRequest(%q) = %q, %t; want %q, %t", tt.raw, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestExtractCommandFromRequest(t *testing.T) {
	event.Init()
	tests := []struct {
		name string
		raw  string
		want string
		ok   bool
	}{
		{"command and arguments", "file.open path=notes ~handle", "file.open", true},
		{"command only", "file.open", "file.open", true},
		{"leading whitespace", " \tfile.open argument", "file.open", true},
		{"unicode whitespace", "file.open\u2003argument", "file.open", true},
		{"empty input", "", "", false},
		{"whitespace only", " \t\n", "", false},
		{"colon in command", "file:open argument", "", false},
		{"colon at start", ":file.open", "", false},
		{"colon at end", "file.open:", "", false},
		{"colon in later field", "file.open path=foo:bar", "file.open", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extractCommandFromRequest(tt.raw)
			if got != tt.want || ok != tt.ok {
				t.Errorf("extractCommandFromRequest(%q) = %q, %t; want %q, %t", tt.raw, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestExtractHandleFromResponse(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
		ok   bool
	}{
		{"response prefix", "~handle:command ok", "handle", true},
		{"hyphenated handle", "~my-handle:command error", "my-handle", true},
		{"only prefix", "~handle:command", "handle", true},
		{"not at start", "text ~handle:command", "", false},
		{"missing colon", "~handle command", "", false},
		{"empty handle", "~:command", "", false},
		{"empty response", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extractHandleFromResponse(tt.raw)
			if got != tt.want || ok != tt.ok {
				t.Errorf("extractHandleFromResponse(%q) = %q, %t; want %q, %t", tt.raw, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestExtractionWitnesses(t *testing.T) {
	event.Init()
	subscriber := &extractionWitnessSubscriber{Base: base.New()}
	event.Subscribe(subscriber)
	defer event.Unsubscribe(subscriber)

	tests := []struct {
		name    string
		extract func(string) (string, bool)
		raw     string
		code    string
		message string
	}{
		{"missing handle", extractHandleFromRequest, "command", "Missing.Mycelium", "request handle not found"},
		{"colon in handle", extractHandleFromRequest, "command ~bad:handle", "Malformed.Mycelium", "request handle contains a colon"},
		{"missing command", extractCommandFromRequest, " \t", "Missing.Mycelium", "request command not found"},
		{"colon in command", extractCommandFromRequest, "bad:command", "Malformed.Mycelium", "request command contains a colon"},
		{"missing response handle", extractHandleFromResponse, "command", "Missing.Mycelium", "response handle not found"},
		{"empty response handle", extractHandleFromResponse, "~:command", "Malformed.Mycelium", "response handle is empty"},
		{"missing response delimiter", extractHandleFromResponse, "~handle command", "Malformed.Mycelium", "response handle missing colon"},
		{"valid handle", extractHandleFromRequest, "command ~handle", "", ""},
		{"valid command", extractCommandFromRequest, "command ~handle", "", ""},
		{"valid response handle", extractHandleFromResponse, "~handle:command", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subscriber.events = nil
			_, ok := tt.extract(tt.raw)
			if tt.message == "" {
				if !ok || len(subscriber.events) != 0 {
					t.Fatalf("successful extraction: ok=%t, witness events=%d", ok, len(subscriber.events))
				}
				return
			}
			if ok || len(subscriber.events) != 1 {
				t.Fatalf("failed extraction: ok=%t, witness events=%d; want false and one event", ok, len(subscriber.events))
			}
			ctx := subscriber.events[0]
			if ctx.Type() != witness_event.WTypeError || !strings.Contains(ctx.Message(), tt.code) || !strings.Contains(ctx.Message(), tt.message) {
				t.Errorf("witness type=%q message=%q; want error containing %q and %q", ctx.Type(), ctx.Message(), tt.code, tt.message)
			}
		})
	}
}
