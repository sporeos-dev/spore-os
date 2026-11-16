package mycelium

import (
	"fmt"
	"reflect"
	"spored/internal/base"
	"spored/internal/cparser"
	"spored/internal/event"
	manifest_event "spored/internal/event/manifest"
	message_event "spored/internal/event/message"
	"spored/internal/manifest"
	"strings"
	"testing"
	"time"
)

func TestPipeSplitPreservesQuotedAndStructuredValues(t *testing.T) {
	raw := "source.read text=\"a | b\" data=[one, {x='|'}] block=<<\ninside | block\n>> ~h | target.write |"
	want := []string{
		"source.read text=\"a | b\" data=[one, {x='|'}] block=<<\ninside | block\n>> ~h",
		"target.write",
		"",
	}
	if got := splitPipe(raw); !reflect.DeepEqual(got, want) {
		t.Fatalf("splitPipe = %#v, want %#v", got, want)
	}
}

type pipeTestEndpoint struct {
	*base.Base
	calls     chan message_event.IRequestContext
	responses chan message_event.IResponseContext
	reply     func(message_event.IRequestContext) string
}

func (p *pipeTestEndpoint) Api() []string {
	return []string{"source.read", "target.write", "target.finish"}
}

func (p *pipeTestEndpoint) Ids() []string { return []string{"caller"} }

func (p *pipeTestEndpoint) OnRequest(ctx message_event.IRequestContext) {
	p.calls <- ctx
	if raw := p.reply(ctx); raw != "" {
		event.Dispatch().Message.Notify(message_event.Raw(raw, ctx.Node()))
	}
}

func (p *pipeTestEndpoint) OnResponse(ctx message_event.IResponseContext) {
	p.responses <- ctx
}

func setupPipeTest(t *testing.T, reply func(message_event.IRequestContext) string) (*Mycelium, *pipeTestEndpoint) {
	t.Helper()
	event.Init()
	m := New()
	m.Open()
	endpoint := &pipeTestEndpoint{
		Base:      base.New(),
		calls:     make(chan message_event.IRequestContext, 8),
		responses: make(chan message_event.IResponseContext, 1),
		reply:     reply,
	}
	event.Subscribe(endpoint)
	inputs := []manifest.Input{{Name: "path"}, {Name: "enabled"}, {Name: "data"}}
	m.commands.OnManifestAnnouncement(manifest_event.New(&manifest.Manifest{
		ID: "source", Api: []manifest.Command{{Name: "read"}},
	}))
	m.commands.OnManifestAnnouncement(manifest_event.New(&manifest.Manifest{
		ID: "target", Api: []manifest.Command{
			{Name: "write", Inputs: &inputs},
			{Name: "finish", Inputs: &inputs},
		},
	}))
	t.Cleanup(func() {
		event.Unsubscribe(endpoint)
		m.Close()
	})
	return m, endpoint
}

func pipeResult(t *testing.T, endpoint *pipeTestEndpoint) message_event.IResponseContext {
	t.Helper()
	select {
	case response := <-endpoint.responses:
		return response
	case <-time.After(3 * time.Second):
		t.Fatal("pipe did not respond")
		return nil
	}
}

func pipeCall(t *testing.T, endpoint *pipeTestEndpoint) message_event.IRequestContext {
	t.Helper()
	select {
	case call := <-endpoint.calls:
		return call
	case <-time.After(3 * time.Second):
		t.Fatal("pipe stage did not run")
		return nil
	}
}

func TestPipeForwardsMatchingFieldsAndPreservesWireValues(t *testing.T) {
	m, endpoint := setupPipeTest(t, func(ctx message_event.IRequestContext) string {
		if ctx.Command() == "source.read" {
			return fmt.Sprintf("~%s:source.read path=\"a \\\"quote\\\" | file\" data=<<\nline | content\n>> enabled ignored=value ok", ctx.Handle())
		}
		return fmt.Sprintf("~%s:target.write result=done ok", ctx.Handle())
	})

	m.OnMessage(message_event.Raw("source.read ~caller-h | target.write timeout=2", "caller"))
	response := pipeResult(t, endpoint)
	first, second := pipeCall(t, endpoint), pipeCall(t, endpoint)
	if first.Cast() != "SPORE.pipe:caller" || first.Origin() != "caller" || second.Cast() != "caller" {
		t.Fatalf("stage identities = %q/%q and %q", first.Cast(), first.Origin(), second.Cast())
	}
	parsed, ok := cparser.ParseAndValidate(second.Message())
	if !ok || parsed.Args["path"] != `a "quote" | file` ||
		!strings.Contains(second.Message(), "data=<<\nline | content\n>>") ||
		!strings.Contains(second.Message(), " enabled") {
		t.Fatalf("forwarded request = %q, parsed = %#v", second.Message(), parsed)
	}
	if strings.Contains(second.Message(), "ignored=") || strings.Contains(second.Message(), "timeout=") {
		t.Fatalf("unexpected forwarded field: %q", second.Message())
	}
	if response.Handle() != "caller-h" || response.Destination() != "caller" ||
		!strings.HasPrefix(response.Message(), "~caller-h:target.write ") {
		t.Fatalf("response = %q, handle=%q destination=%q", response.Message(), response.Handle(), response.Destination())
	}
}

func TestPipeStopsOnFailureCancellationAndDuplicate(t *testing.T) {
	tests := []struct {
		name  string
		reply string
		pipe  string
		want  string
	}{
		{"error", "error code=Generic.Node what=failed", "source.read ~h | target.write", "error code=Generic.Node"},
		{"cancelled", "ok cancelled", "source.read ~h | target.write", "ok cancelled"},
		{"duplicate argument", "path=from_source ok", "source.read ~h | target.write path=explicit", "Collision.Pipe"},
		{"duplicate flag", "enabled ok", "source.read ~h | target.write enabled", "Collision.Pipe"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, endpoint := setupPipeTest(t, func(ctx message_event.IRequestContext) string {
				return fmt.Sprintf("~%s:source.read %s", ctx.Handle(), tt.reply)
			})
			m.OnMessage(message_event.Raw(tt.pipe, "caller"))
			response := pipeResult(t, endpoint)
			if response.Handle() != "h" || !strings.Contains(response.Message(), tt.want) {
				t.Fatalf("response = %q", response.Message())
			}
			pipeCall(t, endpoint)
			select {
			case call := <-endpoint.calls:
				t.Fatalf("unexpected later stage: %s", call.Command())
			default:
			}
		})
	}
}

func TestPipeTimeoutAndMalformedStage(t *testing.T) {
	tests := []struct {
		name  string
		pipe  string
		reply bool
		want  string
	}{
		{"timeout", "source.read timeout=0 ~h | target.write", false, "Timeout.Pipe"},
		{"invalid timeout", "source.read timeout=bad ~h | target.write", false, "Malformed.Pipe"},
		{"later handle", "source.read ~h | target.write ~another", true, "Malformed.Pipe"},
		{"empty final stage", "source.read ~h |", true, "Malformed.Pipe"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, endpoint := setupPipeTest(t, func(ctx message_event.IRequestContext) string {
				if tt.reply {
					return fmt.Sprintf("~%s:source.read ok", ctx.Handle())
				}
				return ""
			})
			m.OnMessage(message_event.Raw(tt.pipe, "caller"))
			if response := pipeResult(t, endpoint); !strings.Contains(response.Message(), tt.want) {
				t.Fatalf("response = %q, want %s", response.Message(), tt.want)
			}
		})
	}
}

func TestPipeTimeoutFalseOverridesDefault(t *testing.T) {
	m, endpoint := setupPipeTest(t, func(ctx message_event.IRequestContext) string {
		if ctx.Command() == "source.read" {
			time.Sleep(60 * time.Millisecond)
		}
		return fmt.Sprintf("~%s:%s ok", ctx.Handle(), ctx.Command())
	})
	m.pending.WithTimeout(20 * time.Millisecond)

	m.OnMessage(message_event.Raw("source.read timeout=false ~h | target.write", "caller"))
	response := pipeResult(t, endpoint)
	if !strings.HasPrefix(response.Message(), "~h:target.write ") || !strings.Contains(response.Message(), " ok") {
		t.Fatalf("response = %q", response.Message())
	}
	first := pipeCall(t, endpoint)
	if strings.Contains(first.Message(), "timeout=") {
		t.Fatalf("timeout was forwarded: %q", first.Message())
	}
	pipeCall(t, endpoint)
}

func TestPipeRejectsInUseHandle(t *testing.T) {
	m, endpoint := setupPipeTest(t, func(message_event.IRequestContext) string { return "" })
	if !m.handles.mapHandle("h", "other") {
		t.Fatal("could not reserve test handle")
	}
	m.OnMessage(message_event.Raw("source.read ~h | target.write", "caller"))
	if response := pipeResult(t, endpoint); !strings.Contains(response.Message(), "HandleInUse.Pipe") {
		t.Fatalf("response = %q", response.Message())
	}
	if destination, ok := m.handles.consumeHandle("h"); !ok || destination != "other" {
		t.Fatalf("original handle mapping changed: %q, %t", destination, ok)
	}
	select {
	case call := <-endpoint.calls:
		t.Fatalf("unexpected stage: %s", call.Command())
	default:
	}
}
