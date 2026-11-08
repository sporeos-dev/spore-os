// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package nodes

import (
	"bufio"
	"net"
	"slices"
	"spored/internal/base"
	"spored/internal/bus"
	"spored/internal/cparser"
	"spored/internal/event"
	witness_event "spored/internal/event/witness"
	"spored/internal/iface"
	"spored/internal/manifest"
	"spored/internal/message"
	"spored/internal/registry"
	sporeerror "spored/internal/utilities/error"
	"strings"
	"testing"
	"time"
)

type routeErrorTestBus struct {
	requestErr  func(iface.Message) *sporeerror.Error
	responseErr func(iface.Message) *sporeerror.Error
}

func TestReturnRoutingErrorReportsDeliveryFailure(t *testing.T) {
	subscriber := &writerWitnessSubscriber{
		Base:   base.New(),
		events: make(chan witness_event.IWitnessContext, 1),
	}
	event.Subscribe(subscriber)
	defer event.Unsubscribe(subscriber)

	node := newNode(&registry.Element{ID: "test.shell"}, &manifest.Manifest{})
	request, ok := message.Request("example.command ~request", node.Id())
	if !ok {
		t.Fatal("invalid request")
	}
	node.returnRoutingError(sporeerror.MissingArg("node", sporeerror.Spore), request)

	select {
	case ctx := <-subscriber.events:
		if ctx.Type() != witness_event.WTypeError ||
			!strings.Contains(ctx.Message(), "failed to return routing error") ||
			!strings.Contains(ctx.Message(), "node not connected") {
			t.Errorf("unexpected witness event: %q", ctx.Message())
		}
	default:
		t.Fatal("expected witness event for failed error delivery")
	}
}

func (*routeErrorTestBus) Register(bus.INode)   {}
func (*routeErrorTestBus) Unregister(bus.INode) {}
func (*routeErrorTestBus) Broadcast(iface.Message) *sporeerror.Error {
	return nil
}
func (*routeErrorTestBus) Pipe(iface.Message, bus.INode) {}
func (b *routeErrorTestBus) Request(msg iface.Message) *sporeerror.Error {
	return b.requestErr(msg)
}
func (b *routeErrorTestBus) Response(msg iface.Message) *sporeerror.Error {
	return b.responseErr(msg)
}

func TestListenerReturnsRoutingErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		handle      string
		command     string
		code        string
		what        string
		argument    string
		requestErr  func(iface.Message) *sporeerror.Error
		responseErr func(iface.Message) *sporeerror.Error
	}{
		{
			name:     "spawn missing node",
			input:    "dev.sporeos.SPORE.node.spawn ~missing-node\n",
			handle:   "missing-node",
			command:  "dev.sporeos.SPORE.node.spawn",
			code:     "Missing.Spore",
			what:     "missing argument",
			argument: "node",
			requestErr: func(msg iface.Message) *sporeerror.Error {
				return sporeerror.MissingArg("node", sporeerror.Spore).WithMessage(msg)
			},
		},
		{
			name:    "response routing failure",
			input:   "~unknown:example.command ok\n",
			handle:  "unknown",
			command: "example.command",
			code:    "Missing.Bus",
			what:    "handle not mapped",
			responseErr: func(msg iface.Message) *sporeerror.Error {
				return sporeerror.New(sporeerror.Missing, sporeerror.Bus, "handle not mapped")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subscriber := &writerWitnessSubscriber{
				Base:   base.New(),
				events: make(chan witness_event.IWitnessContext, 4),
			}
			event.Subscribe(subscriber)
			defer event.Unsubscribe(subscriber)

			server, client := net.Pipe()
			node := newNode(&registry.Element{ID: "test.shell"}, &manifest.Manifest{})
			node.bus = &routeErrorTestBus{requestErr: tt.requestErr, responseErr: tt.responseErr}
			node.conn = server
			node.reader = bufio.NewReader(server)
			node.writer = newWriter(server, bufio.NewWriter(server), node.Id())

			done := make(chan struct{})
			go func() {
				node.listen()
				close(done)
			}()
			defer func() {
				client.Close()
				select {
				case <-done:
				case <-time.After(time.Second):
					t.Error("listener did not stop")
				}
			}()

			if err := client.SetDeadline(time.Now().Add(time.Second)); err != nil {
				t.Fatal(err)
			}
			if _, err := client.Write([]byte(tt.input)); err != nil {
				t.Fatalf("send message: %v", err)
			}
			raw, err := bufio.NewReader(client).ReadString('\n')
			if err != nil {
				t.Fatalf("read routing error: %v", err)
			}
			reply, ok := cparser.Validate(raw)
			if !ok {
				t.Fatalf("invalid error response: %q", raw)
			}
			if reply.Args["code"] != tt.code {
				t.Errorf("error code = %q, want %q", reply.Args["code"], tt.code)
			}
			if tt.argument != "" && reply.Args["argument"] != tt.argument {
				t.Errorf("argument = %q, want %q", reply.Args["argument"], tt.argument)
			}
			if reply.Handle != tt.handle || reply.Command != tt.command ||
				reply.Args["what"] != tt.what || !slices.Contains(reply.Flags, "error") ||
				slices.Contains(reply.Flags, "ok") {
				t.Errorf("unexpected response: %q", raw)
			}
			for _, want := range []witness_event.WType{witness_event.WTypeIncoming, witness_event.WTypeOutgoing} {
				select {
				case ctx := <-subscriber.events:
					if ctx.Type() != want {
						t.Errorf("witness type = %q, want %q", ctx.Type(), want)
					}
				case <-time.After(time.Second):
					t.Fatalf("missing witness event %q", want)
				}
			}
		})
	}
}
