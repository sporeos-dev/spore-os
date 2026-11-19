package mycelium

import (
	"fmt"
	"spored/internal/base"
	"spored/internal/cparser"
	"spored/internal/event"
	manifest_event "spored/internal/event/manifest"
	message_event "spored/internal/event/message"
	sync_event "spored/internal/event/sync"
	"spored/internal/manifest"
	"strings"
	"testing"
	"time"
)

func setupPermissionGrant(t *testing.T, m *Mycelium) {
	t.Helper()
	spore := &manifest.Manifest{
		ID: "dev.sporeos.SPORE", Trust: manifest.System,
		Api: []manifest.Command{{Name: "permission.grant"}},
	}
	m.commands.OnManifestAnnouncement(manifest_event.New(spore))
	m.allowance.OnManifestAnnouncement(manifest_event.New(spore))
}

func grantPermission(t *testing.T, m *Mycelium, capability string) {
	t.Helper()
	raw := fmt.Sprintf("dev.sporeos.SPORE.permission.grant ~grant node=caller capability=%s", capability)
	request := sync_event.Request(raw, "dev.sporeos.SPORE")
	m.OnSynchronizedRequest(request)
	response := request.Response()
	if response == nil || !strings.Contains(response.Message(), " ok") {
		t.Fatalf("grant response = %v", response)
	}
}

func TestSynchronizedRequestChecksOriginalCaller(t *testing.T) {
	m, endpoint := setupPipeTest(t, func(ctx message_event.IRequestContext) string {
		return fmt.Sprintf("~%s:%s ok", ctx.Handle(), ctx.Command())
	})
	m.allowance.OnManifestWithdrawal(manifest_event.New(&manifest.Manifest{
		ID: "caller", Trust: manifest.System,
	}))
	setupPermissionGrant(t, m)

	raw := "read request.permission ~sync"
	for _, cast := range []string{"SPORE.pipe:caller", "dev.sporeos.SPORE:caller"} {
		denied := sync_event.Request("read ~sync", cast)
		m.OnSynchronizedRequest(denied)
		if denied.Response() == nil || !strings.Contains(denied.Response().Message(), "NotPermitted.Mycelium") {
			t.Fatalf("denied response for %q = %v", cast, denied.Response())
		}
	}
	select {
	case call := <-endpoint.calls:
		t.Fatalf("denied request was forwarded: %s", call.Command())
	default:
	}

	grantPermission(t, m, "source.read")
	allowed := sync_event.Request(raw, "SPORE.pipe:caller")
	m.OnSynchronizedRequest(allowed)
	if allowed.Response() == nil || !strings.Contains(allowed.Response().Message(), " ok") {
		t.Fatalf("allowed response = %v", allowed.Response())
	}
	call := pipeCall(t, endpoint)
	if call.Origin() != "caller" || call.Cast() != "SPORE.pipe:caller" {
		t.Fatalf("request identity = %q/%q", call.Cast(), call.Origin())
	}
	if strings.Contains(call.Message(), "request.permission") {
		t.Fatalf("hub flag was forwarded: %q", call.Message())
	}
}

func TestPipeChecksEachStage(t *testing.T) {
	m, endpoint := setupPipeTest(t, func(ctx message_event.IRequestContext) string {
		return fmt.Sprintf("~%s:%s ok", ctx.Handle(), ctx.Command())
	})
	m.allowance.OnManifestWithdrawal(manifest_event.New(&manifest.Manifest{
		ID: "caller", Trust: manifest.System,
	}))
	setupPermissionGrant(t, m)

	m.OnMessage(message_event.Raw("read ~h | write", "caller"))
	if response := pipeResult(t, endpoint); !strings.Contains(response.Message(), "NotPermitted.Mycelium") {
		t.Fatalf("first stage denial = %q", response.Message())
	}
	select {
	case call := <-endpoint.calls:
		t.Fatalf("denied first stage was forwarded: %s", call.Command())
	default:
	}

	grantPermission(t, m, "source.read")
	m.OnMessage(message_event.Raw("read ~h | write", "caller"))
	if response := pipeResult(t, endpoint); !strings.Contains(response.Message(), "NotPermitted.Mycelium") {
		t.Fatalf("second stage denial = %q", response.Message())
	}
	if call := pipeCall(t, endpoint); call.Command() != "source.read" || call.Origin() != "caller" {
		t.Fatalf("first stage = %q from %q", call.Command(), call.Origin())
	}
	select {
	case call := <-endpoint.calls:
		t.Fatalf("denied second stage was forwarded: %s", call.Command())
	default:
	}

	grantPermission(t, m, "target.write")
	m.OnMessage(message_event.Raw("read ~h | write", "caller"))
	if response := pipeResult(t, endpoint); !strings.Contains(response.Message(), " ok") {
		t.Fatalf("allowed pipe response = %q", response.Message())
	}
	first, second := pipeCall(t, endpoint), pipeCall(t, endpoint)
	if first.Command() != "source.read" || second.Command() != "target.write" {
		t.Fatalf("pipe stages = %q, %q", first.Command(), second.Command())
	}
}

type permissionFlowEndpoint struct {
	*base.Base
	calls     chan message_event.IRequestContext
	responses chan message_event.IResponseContext
}

func (p *permissionFlowEndpoint) Api() []string {
	return []string{"source.read", "target.write", "dev.sporeos.shell.confirm"}
}

func (p *permissionFlowEndpoint) Ids() []string { return []string{"caller"} }

func (p *permissionFlowEndpoint) OnRequest(ctx message_event.IRequestContext) {
	p.calls <- ctx
	flags := "ok"
	if ctx.Command() == "dev.sporeos.shell.confirm" {
		flags = "ok yes"
	}
	raw := fmt.Sprintf("~%s:%s %s", ctx.Handle(), ctx.Command(), flags)
	event.Dispatch().Message.Notify(message_event.Raw(raw, ctx.Node()))
}

func (p *permissionFlowEndpoint) OnResponse(ctx message_event.IResponseContext) {
	p.responses <- ctx
}

func TestPermissionRequestWorksAcrossRoutes(t *testing.T) {
	for _, route := range []string{"synchronized", "asynchronous", "pipe"} {
		t.Run(route, func(t *testing.T) {
			event.Init()
			m := New()
			m.Open()
			endpoint := &permissionFlowEndpoint{
				Base: base.New(), calls: make(chan message_event.IRequestContext, 8),
				responses: make(chan message_event.IResponseContext, 1),
			}
			event.Subscribe(endpoint)
			defer event.Unsubscribe(endpoint)
			defer m.Close()

			manifests := []*manifest.Manifest{
				{ID: "source", Api: []manifest.Command{{Name: "read"}}},
				{ID: "target", Api: []manifest.Command{{Name: "write"}}},
				{ID: "dev.sporeos.shell", Api: []manifest.Command{{Name: "confirm"}}},
				{ID: "dev.sporeos.SPORE", Trust: manifest.System, Api: []manifest.Command{{Name: "permission.request"}}},
			}
			for _, man := range manifests {
				m.commands.OnManifestAnnouncement(manifest_event.New(man))
				m.allowance.OnManifestAnnouncement(manifest_event.New(man))
			}

			var response message_event.IResponseContext
			if route == "pipe" {
				m.OnMessage(message_event.Raw("read request.permission ~h | write request.permission", "caller"))
				select {
				case response = <-endpoint.responses:
				case <-time.After(3 * time.Second):
					t.Fatal("pipe did not respond")
				}
			} else if route == "asynchronous" {
				m.OnMessage(message_event.Raw("read request.permission ~h", "caller"))
				select {
				case response = <-endpoint.responses:
				case <-time.After(3 * time.Second):
					t.Fatal("request did not respond")
				}
			} else {
				request := sync_event.Request("read request.permission ~h", "caller")
				m.OnSynchronizedRequest(request)
				response = request.Response()
			}
			if response == nil || !strings.Contains(response.Message(), " ok") {
				t.Fatalf("response = %v", response)
			}

			want := []string{"dev.sporeos.shell.confirm", "source.read"}
			if route == "pipe" {
				want = append(want, "dev.sporeos.shell.confirm", "target.write")
			}
			for _, command := range want {
				select {
				case call := <-endpoint.calls:
					if call.Command() != command {
						t.Fatalf("command = %q, want %q", call.Command(), command)
					}
					if command != "dev.sporeos.shell.confirm" {
						if call.Origin() != "caller" || strings.Contains(call.Message(), "request.permission") {
							t.Fatalf("forwarded request = %q from %q", call.Message(), call.Origin())
						}
					} else {
						parsed, ok := cparser.ParseAndValidate(call.Message())
						if !ok || !strings.Contains(parsed.Args["body"], "[caller]") {
							t.Fatalf("confirmation did not identify caller: %q", call.Message())
						}
					}
				default:
					t.Fatalf("missing call to %s", command)
				}
			}
		})
	}
}
