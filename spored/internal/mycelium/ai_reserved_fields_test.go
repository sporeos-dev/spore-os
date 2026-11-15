package mycelium

import (
	"spored/internal/base"
	"spored/internal/cparser"
	"spored/internal/event"
	manifest_event "spored/internal/event/manifest"
	message_event "spored/internal/event/message"
	sync_event "spored/internal/event/sync"
	witness_event "spored/internal/event/witness"
	"spored/internal/manifest"
	sporeerror "spored/internal/utilities/error"
	"strings"
	"testing"
)

type reservedFieldSubscriber struct {
	*base.Base
	requests   []message_event.IRequestContext
	responses  []message_event.IResponseContext
	broadcasts []message_event.IBroadcastContext
	witnesses  []witness_event.IWitnessContext
}

func (s *reservedFieldSubscriber) Api() []string { return []string{"node.time"} }
func (s *reservedFieldSubscriber) Ids() []string { return []string{"caller", "subscriber"} }
func (s *reservedFieldSubscriber) OnRequest(ctx message_event.IRequestContext) {
	s.requests = append(s.requests, ctx)
}
func (s *reservedFieldSubscriber) OnResponse(ctx message_event.IResponseContext) {
	s.responses = append(s.responses, ctx)
}
func (s *reservedFieldSubscriber) OnBroadcast(ctx message_event.IBroadcastContext) {
	s.broadcasts = append(s.broadcasts, ctx)
}
func (s *reservedFieldSubscriber) OnWitness(ctx witness_event.IWitnessContext) {
	s.witnesses = append(s.witnesses, ctx)
}

func TestReservedFieldsAtTopLevel(t *testing.T) {
	event.Init()
	tests := []struct {
		name  string
		raw   string
		check func(string) bool
		want  bool
	}{
		{"request without cast", "node.time ~h", ensureNoCast, true},
		{"request cast", "node.time cast=spoof ~h", ensureNoCast, false},
		{"request cast with tab", "node.time\tcast=spoof ~h", ensureNoCast, false},
		{"request empty cast", "node.time cast= ~h", ensureNoCast, false},
		{"duplicate cast", "node.time cast=first cast=second ~h", ensureNoCast, false},
		{"quoted cast key", `node.time "cast"=spoof ~h`, ensureNoCast, false},  
		{"quoted value", `node.time text="cast=allowed" ~h`, ensureNoCast, true},
		{"object value", `node.time data={cast=allowed} ~h`, ensureNoCast, true},
		{"array value", `node.time data=[cast=allowed] ~h`, ensureNoCast, true},
		{"block value", "node.time data=<<\ncast=allowed\n>> ~h", ensureNoCast, true},
		{"parser sees cast after embedded block delimiter", "node.time data=<<\ntext >> cast=ambiguous\n>> ~h", ensureNoCast, false},
		{"cast after block", "node.time data=<<\ntext\n>> cast=spoof ~h", ensureNoCast, false},
		{"broadcast cast", "publish node.tick cast=spoof", ensureNoCast, false},
		{"broadcast quoted value", `publish node.tick text="cast=allowed"`, ensureNoCast, true},
		{"response capture", "~h:node.time ok capture=spoof", ensureNoCapture, false},
		{"response capture with tab", "~h:node.time ok\tcapture=spoof", ensureNoCapture, false},
		{"response quoted value", `~h:node.time ok text="capture=allowed"`, ensureNoCapture, true},
		{"response after block", "~h:node.time data=<<\ntext\n>> capture=spoof ok", ensureNoCapture, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.check(tt.raw); got != tt.want {
				t.Errorf("check(%q) = %t, want %t", tt.raw, got, tt.want)
			}
		})
	}
}

func TestReservedFieldsRejectedBeforeRouting(t *testing.T) {
	event.Init()
	m := New()
	m.Open()
	defer m.Close()
	subscriber := &reservedFieldSubscriber{Base: base.New()}
	event.Subscribe(subscriber)
	defer event.Unsubscribe(subscriber)

	m.commands.OnManifestAnnouncement(manifest_event.New(&manifest.Manifest{
		ID: "node", Api: []manifest.Command{{Name: "time"}},
	}))
	m.topics.abbreviations["node.tick"] = []string{"node.tick"}
	m.topics.publishers["node.tick"] = "node"
	m.topics.subscribers["node.tick"] = []string{"subscriber"}

	m.OnMessage(message_event.Raw("node.time cast=spoof ~bad", "caller"))
	if len(subscriber.requests) != 0 || len(subscriber.responses) != 1 {
		t.Fatalf("spoofed request: requests=%d responses=%d", len(subscriber.requests), len(subscriber.responses))
	}
	errorResponse := subscriber.responses[0].Message()
	parsed, ok := cparser.ParseAndValidate(errorResponse)
	if !ok || parsed.Handle != "bad" || parsed.Args["code"] != "ReservedLanguage.Mycelium" ||
		parsed.Args["what"] != "cast is hub-reserved" {
		t.Fatalf("unexpected error response %q: %#v", errorResponse, parsed)
	}
	if !hasReservedWitness(subscriber.witnesses) {
		t.Fatal("spoofed request was not witnessed")
	}

	m.OnMessage(message_event.Raw(`node.time text="cast=allowed" ~good`, "caller"))
	if len(subscriber.requests) != 1 {
		t.Fatalf("legitimate request was not forwarded: %d", len(subscriber.requests))
	}
	m.OnMessage(message_event.Raw("~good:node.time ok capture=spoof", "node"))
	if len(subscriber.responses) != 1 {
		t.Fatal("spoofed response was forwarded")
	}
	m.OnMessage(message_event.Raw("~good:node.time ok", "node"))
	if len(subscriber.responses) != 2 || subscriber.responses[1].Handle() != "good" {
		t.Fatal("legitimate response was not forwarded after spoofed response")
	}

	m.OnMessage(message_event.Raw("publish node.tick cast=spoof", "node"))
	m.OnMessage(message_event.Raw(`publish node.tick text="cast=allowed"`, "node"))
	if len(subscriber.broadcasts) != 1 {
		t.Fatalf("broadcasts delivered = %d, want 1", len(subscriber.broadcasts))
	}

	syncRequest := sync_event.Request("node.time cast=spoof ~sync", "caller")
	m.OnSynchronizedRequest(syncRequest)
	if syncRequest.Response() == nil || !strings.Contains(syncRequest.Response().Message(), "ReservedLanguage.Mycelium") {
		t.Fatal("synchronized spoofed request was not rejected")
	}
}

func hasReservedWitness(events []witness_event.IWitnessContext) bool {
	for _, ctx := range events {
		if ctx.Type() == witness_event.WTypeError && strings.Contains(ctx.Message(), "ReservedLanguage.Mycelium") {
			return true
		}
	}
	return false
}

func TestRequestHandleSkipsBlockContent(t *testing.T) {
	event.Init()
	raw := "node.time data=<<\n~not-the-handle\n>> cast=spoof ~actual"
	handle, ok := extractHandleFromRequest(raw)
	if !ok || handle != "actual" {
		t.Fatalf("handle = %q, %t; want actual, true", handle, ok)
	}
}

func TestResponseErrorQuotesDescription(t *testing.T) {
	event.Init()
	description := "message with \"quotes\", \\slashes, and\nnewlines"
	response := message_event.ResponseError(sporeerror.Malformed, sporeerror.Mycelium, description, "h", "node.time", "caller")
	parsed, ok := cparser.ParseAndValidate(response.Message())
	if !ok || parsed.Args["what"] != description {
		t.Fatalf("response %q parsed as %#v", response.Message(), parsed)
	}
}
