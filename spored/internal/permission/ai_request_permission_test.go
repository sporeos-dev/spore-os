package permission

import (
	"fmt"
	"spored/internal/base"
	"spored/internal/cparser"
	"spored/internal/event"
	message_event "spored/internal/event/message"
	sync_event "spored/internal/event/sync"
	"testing"
	"time"
)

type permissionRequestProbe struct {
	*base.Base
	t     *testing.T
	calls int
}

func (p *permissionRequestProbe) OnSynchronizedRequest(ctx sync_event.ISynchronizedRequestContext) {
	p.calls++
	request, ok := cparser.ParseAndValidate(ctx.Message())
	if !ok {
		p.t.Fatal("invalid permission request")
	}
	raw := fmt.Sprintf("~%s:%s ok no", request.Handle, request.Command)
	ctx.Respond(message_event.Response(raw, "dev.sporeos.SPORE", request.Handle, ctx.Origin()))
}

func TestCanRequestsPermissionOnlyWithoutTokenArgument(t *testing.T) {
	tests := []struct {
		name  string
		raw   string
		calls int
	}{
		{"token text in quoted value", `node.call note="permission.token=other" request.permission ~h`, 1},
		{"token text in other argument", `node.call other.permission.token=other request.permission ~h`, 1},
		{"empty token argument", `node.call permission.token="" request.permission ~h`, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event.Init()
			probe := &permissionRequestProbe{Base: base.New(), t: t}
			event.Subscribe(probe)
			defer event.Unsubscribe(probe)

			if New().Can("caller", "node.call", tt.raw) {
				t.Fatal("permission should not be granted")
			}
			if probe.calls != tt.calls {
				t.Errorf("permission requests = %d, want %d", probe.calls, tt.calls)
			}
		})
	}
}

type approvingPermissionRequestProbe struct {
	*base.Base
	t         *testing.T
	allowance *Allowance
}

func (p *approvingPermissionRequestProbe) OnSynchronizedRequest(ctx sync_event.ISynchronizedRequestContext) {
	request, ok := cparser.ParseAndValidate(ctx.Message())
	if !ok {
		p.t.Fatal("invalid permission request")
	}

	if !p.allowance.mu.TryLock() {
		p.t.Error("allowance lock held while requesting permission")
		raw := fmt.Sprintf("~%s:%s ok no", request.Handle, request.Command)
		ctx.Respond(message_event.Response(raw, "dev.sporeos.SPORE", request.Handle, ctx.Origin()))
		return
	}
	p.allowance.tokens = append(p.allowance.tokens, temporaryGrant{
		node: "caller", capability: "node.call", when: time.Now(), token: "valid",
	})
	p.allowance.mu.Unlock()

	raw := fmt.Sprintf("~%s:%s ok permission.token=valid", request.Handle, request.Command)
	ctx.Respond(message_event.Response(raw, "dev.sporeos.SPORE", request.Handle, ctx.Origin()))
}

func TestCanUnlocksDuringPermissionRequest(t *testing.T) {
	event.Init()
	a := New()
	probe := &approvingPermissionRequestProbe{Base: base.New(), t: t, allowance: a}
	event.Subscribe(probe)
	defer event.Unsubscribe(probe)

	if !a.Can("caller", "node.call", "node.call request.permission ~h") {
		t.Fatal("approved request was not authorized")
	}
	if a.Can("caller", "node.call", "node.call permission.token=valid ~h") {
		t.Fatal("approved token was not consumed")
	}
}
