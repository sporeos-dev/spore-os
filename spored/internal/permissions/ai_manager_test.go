// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package permissions

import (
	"fmt"
	"spored/internal/bus"
	"spored/internal/iface"
	"spored/internal/manifest"
	"spored/internal/message"
	"spored/internal/utilities/await"
	sporeerror "spored/internal/utilities/error"
	"strings"
	"sync"
	"testing"
	"time"
)

type confirmationHyphae struct {
	confirmed bool
	calls     int
	node      string
	command   string
}

type developerInstallConfirmationBus struct {
	manager  *Manager
	requests []iface.Message
	yes      bool
}

func (b *developerInstallConfirmationBus) Register(bus.INode)   {}
func (b *developerInstallConfirmationBus) Unregister(bus.INode) {}

func (b *developerInstallConfirmationBus) Request(request iface.Message) *sporeerror.Error {
	b.requests = append(b.requests, request)
	response, ok := message.Response(
		fmt.Sprintf("~%s:dev.sporeos.shell.confirm yes ok", request.Handle()),
		"dev.sporeos.shell")
	if !ok {
		panic("failed to create shell confirmation response")
	}
	if !b.yes {
		response, ok = message.Response(
			fmt.Sprintf("~%s:dev.sporeos.shell.confirm ok", request.Handle()),
			"dev.sporeos.shell")
		if !ok {
			panic("failed to create shell denial response")
		}
	}
	return b.manager.Receive(response)
}

func (h *confirmationHyphae) Confirm(node string, command string) (bool, *sporeerror.Error) {
	h.calls++
	h.node = node
	h.command = command
	return h.confirmed, nil
}

func TestDeveloperInstallationRequiresShellWarningBeforeHyphaeConfirmation(t *testing.T) {
	hyphae := &confirmationHyphae{confirmed: true}
	manager := &Manager{
		pending:    await.New(sporeerror.Permission),
		allowances: make(map[string][]string),
		hyphae:     hyphae,
	}
	testBus := &developerInstallConfirmationBus{manager: manager, yes: true}
	manager.bus = testBus

	granted, err := manager.AuthorizeInstallation(
		"dev.sporeos.shell",
		"dev.sporeos.example",
		"dev.sporeos.SPORE.node.install",
		true)
	if err != nil || !granted {
		t.Fatalf("developer installation authorization: granted=%t err=%v", granted, err)
	}
	if len(testBus.requests) != 1 {
		t.Fatalf("shell confirmation count = %d, want 1", len(testBus.requests))
	}
	if got := testBus.requests[0].ArgIf("body", ""); got == "" ||
		!strings.Contains(got, "Developer trust bypasses checksum verification") {
		t.Fatalf("developer warning body = %q", got)
	}
	if hyphae.calls != 1 || hyphae.node != "dev.sporeos.shell" ||
		hyphae.command != "dev.sporeos.SPORE.node.install" {
		t.Fatalf("unexpected Hyphae confirmation: calls=%d node=%q command=%q",
			hyphae.calls, hyphae.node, hyphae.command)
	}
}

func TestDeveloperInstallationDenialSkipsHyphaeConfirmation(t *testing.T) {
	hyphae := &confirmationHyphae{confirmed: true}
	manager := &Manager{
		pending:    await.New(sporeerror.Permission),
		allowances: make(map[string][]string),
		hyphae:     hyphae,
	}
	testBus := &developerInstallConfirmationBus{manager: manager}
	manager.bus = testBus

	granted, err := manager.AuthorizeInstallation(
		"dev.sporeos.shell",
		"dev.sporeos.example",
		"dev.sporeos.SPORE.node.install",
		true)
	if err != nil || granted {
		t.Fatalf("developer installation denial: granted=%t err=%v", granted, err)
	}
	if hyphae.calls != 0 {
		t.Fatalf("Hyphae confirmation calls = %d, want 0 after shell denial", hyphae.calls)
	}
}

// TestHandleUnique guards against the index++ race in handle generation.
func TestHandleUnique(t *testing.T) {
	m := &Manager{}

	const goroutines = 500
	handles := make([]string, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer wg.Done()
			handles[i] = m.handle()
		}(i)
	}
	wg.Wait()

	seen := make(map[string]bool, goroutines)
	for _, handle := range handles {
		if seen[handle] {
			t.Fatalf("duplicate handle generated: %q", handle)
		}
		seen[handle] = true
	}
}

func TestDeveloperNodeRequiresExplicitPermission(t *testing.T) {
	m := &Manager{
		allowances: make(map[string][]string),
	}
	m.RegisterNode(&manifest.Manifest{
		ID:    "dev.sporeos.example",
		Trust: manifest.Developer,
	})

	if allowed, err := m.Can(testRequest(t, "dev.sporeos.example"), "filesystem.write"); err != nil || allowed {
		t.Fatal("developer node received an implicit capability grant")
	}

	m.allowances["dev.sporeos.example"] = []string{"filesystem.write"}
	if allowed, err := m.Can(testRequest(t, "dev.sporeos.example"), "filesystem.write"); err != nil || !allowed {
		t.Fatal("developer node was denied an explicit capability grant")
	}
}

func TestProtectedCapabilityRequiresHubInternalCaller(t *testing.T) {
	m := &Manager{
		allowances: make(map[string][]string),
	}
	m.RegisterNode(&manifest.Manifest{
		ID:    "dev.sporeos.system",
		Trust: manifest.System,
	})
	m.RegisterNode(&manifest.Manifest{
		ID: "dev.sporeos.target",
		Api: []manifest.Command{{
			Name: "internal",
			Risk: manifest.Protected,
		}},
	})

	capability := "dev.sporeos.target.internal"
	if allowed, err := m.Can(testRequest(t, "dev.sporeos.system"), capability); err != nil || allowed {
		t.Fatalf("system caller allowed protected capability: allowed=%t err=%v", allowed, err)
	}
	if allowed, err := m.Can(testRequest(t, protectedCaller+":dev.sporeos.user"), capability); err != nil || !allowed {
		t.Fatalf("hub-internal caller denied protected capability: allowed=%t err=%v", allowed, err)
	}
	if allowed, err := m.Can(testRequest(t, "dev.sporeos.user:"+protectedCaller), capability); err != nil || allowed {
		t.Fatalf("protected capability used origin instead of actor: allowed=%t err=%v", allowed, err)
	}
}

func TestPermissionManagerCanRequestShellConfirmation(t *testing.T) {
	m := &Manager{
		allowances: make(map[string][]string),
	}

	if allowed, err := m.Can(testRequest(t, permissionCaller), permissionConfirmationCapability); err != nil || !allowed {
		t.Fatalf("permission manager was denied shell confirmation: allowed=%t err=%v", allowed, err)
	}
	if allowed, err := m.Can(testRequest(t, permissionCaller), "dev.sporeos.shell.other"); err != nil || allowed {
		t.Fatalf("permission manager was allowed an unrelated capability: allowed=%t err=%v", allowed, err)
	}
	if allowed, err := m.Can(testRequest(t, protectedCaller), hyphaeConfirmationCapability); err != nil || !allowed {
		t.Fatalf("hyphae was denied its confirmation capability: allowed=%t err=%v", allowed, err)
	}
	if allowed, err := m.Can(testRequest(t, protectedCaller), "dev.sporeos.HYPHAE.other"); err != nil || allowed {
		t.Fatalf("hyphae was allowed an unrelated capability: allowed=%t err=%v", allowed, err)
	}
}

func TestPermissionPermitIsBoundToOriginCapabilityAndOneUse(t *testing.T) {
	token := "permit"
	m := &Manager{
		allowances: make(map[string][]string),
		permits: map[string]permit{
			token: {
				origin:     "dev.sporeos.requester",
				capability: "dev.sporeos.store.get",
				expiresAt:  time.Now().Add(permissionTokenLifetime),
			},
		},
	}

	if allowed, err := m.Can(
		testRequestWithToken(t, "dev.sporeos.requester", token),
		"dev.sporeos.store.get"); err != nil || !allowed {
		t.Fatalf("permit authorization: allowed=%t err=%v", allowed, err)
	}
	if allowed, err := m.Can(
		testRequestWithToken(t, "dev.sporeos.requester", token),
		"dev.sporeos.store.get"); err != nil || allowed {
		t.Fatalf("permit was reusable: allowed=%t err=%v", allowed, err)
	}
}

func TestAdminCapabilityRequiresFreshHyphaeConfirmation(t *testing.T) {
	hyphae := &confirmationHyphae{confirmed: true}
	m := &Manager{
		allowances: make(map[string][]string),
		hyphae:     hyphae,
	}
	m.RegisterNode(&manifest.Manifest{
		ID:    "dev.sporeos.system",
		Trust: manifest.System,
	})
	m.RegisterNode(&manifest.Manifest{
		ID: "dev.sporeos.target",
		Api: []manifest.Command{{
			Name: "change",
			Risk: manifest.Admin,
		}},
	})

	capability := "dev.sporeos.target.change"
	allowed, err := m.Can(testRequest(t, "SPORE.pipe:dev.sporeos.system"), capability)
	if err != nil || !allowed {
		t.Fatalf("admin confirmation result: allowed=%t err=%v", allowed, err)
	}
	if hyphae.calls != 1 || hyphae.node != "dev.sporeos.system" || hyphae.command != capability {
		t.Fatalf("unexpected confirmation: calls=%d node=%q command=%q",
			hyphae.calls, hyphae.node, hyphae.command)
	}

	hyphae.confirmed = false
	allowed, err = m.Can(testRequest(t, "SPORE.pipe:dev.sporeos.system"), capability)
	if err != nil || allowed {
		t.Fatalf("denied admin confirmation result: allowed=%t err=%v", allowed, err)
	}
	if hyphae.calls != 2 {
		t.Fatalf("confirmation calls = %d, want 2", hyphae.calls)
	}
}

func testRequest(t *testing.T, cast string) iface.Message {
	t.Helper()

	msg, ok := message.Request("dev.sporeos.target.run ~test", cast)
	if !ok {
		t.Fatal("Request() failed")
	}
	return msg
}

func testRequestWithToken(t *testing.T, cast string, token string) iface.Message {
	t.Helper()

	msg, ok := message.Request(
		"dev.sporeos.target.run permission.token="+token+" ~test",
		cast)
	if !ok {
		t.Fatal("Request() failed")
	}
	return msg
}
