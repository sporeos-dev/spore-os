// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package nodes

import (
	"bufio"
	"net"
	"spored/internal/bus"
	"spored/internal/iface"
	"spored/internal/manifest"
	"spored/internal/registry"
	sporeerror "spored/internal/utilities/error"
	"strings"
	"testing"
	"time"
)

type permissionTestBus struct {
	requests chan iface.Message
}

func (b *permissionTestBus) Register(bus.INode)                       {}
func (b *permissionTestBus) Unregister(bus.INode)                     {}
func (b *permissionTestBus) Response(iface.Message) *sporeerror.Error { return nil }
func (b *permissionTestBus) Broadcast(iface.Message) *sporeerror.Error {
	return nil
}
func (b *permissionTestBus) Pipe(iface.Message, bus.INode) {}
func (b *permissionTestBus) Witness(iface.Message)         {}

func (b *permissionTestBus) Request(msg iface.Message) *sporeerror.Error {
	b.requests <- msg
	return nil
}

type permissionTestPermitter struct{}

func (permissionTestPermitter) Can(_ string, capability string) bool {
	return capability == "alias.context.add"
}

func (permissionTestPermitter) RegisterNode(*manifest.Manifest) {}

func TestNodeKeepsListeningAfterPermissionDenied(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()

	bus := &permissionTestBus{requests: make(chan iface.Message, 1)}
	node := newNode(
		&registry.Element{ID: "dev.sporeos.foundry.alias"},
		&manifest.Manifest{},
	)
	node.bus = bus
	node.permitter = permissionTestPermitter{}
	node.conn = server
	node.reader = bufio.NewReader(server)
	node.writer = newWriter(server, bufio.NewWriter(server))

	done := make(chan struct{})
	go func() {
		node.listen()
		close(done)
	}()

	if _, err := client.Write([]byte("dev.sporeos.SPORE.state.retrieve key=state ~denied\n")); err != nil {
		t.Fatalf("send denied request: %v", err)
	}

	if err := client.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	reply, err := bufio.NewReader(client).ReadString('\n')
	if err != nil {
		t.Fatalf("read permission-denied response: %v", err)
	}
	if !strings.Contains(reply, "~denied") || !strings.Contains(reply, "error") {
		t.Fatalf("unexpected permission-denied response: %q", reply)
	}

	if _, err := client.Write([]byte("alias.context.add key=test value=value ~allowed\n")); err != nil {
		t.Fatalf("send permitted request: %v", err)
	}

	select {
	case request := <-bus.requests:
		if request.Handle() != "allowed" {
			t.Fatalf("request handle = %q, want %q", request.Handle(), "allowed")
		}
	case <-time.After(time.Second):
		t.Fatal("permitted request was not routed after permission denial")
	}

	client.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("node listener did not stop after the connection closed")
	}
}
