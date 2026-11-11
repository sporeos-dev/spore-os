package mycelium

import (
	"spored/internal/base"
	"spored/internal/event"
	message_event "spored/internal/event/message"
	sync_event "spored/internal/event/sync"
	"spored/internal/message"
	"spored/internal/utilities/await"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"spored/internal/witness"
	"strings"
	"time"
)

const defaultTimeout = 12 * time.Second

type Mycelium struct {
	*base.Base
	handles  *handleMap
	commands *commandMap
	topics   *topics
	pending *await.Pending
}

func New() *Mycelium {
	return &Mycelium{
		Base:     base.New().SetName("mycelium"),
		handles:  newHandleMap(),
		commands: newCommandMap(),
		topics: newTopics(),
		pending: await.New(error.Mycelium).WithTimeout(defaultTimeout),
	}
}

func (m *Mycelium) Open() {
	event.Subscribe(m)
	event.Subscribe(m.handles)
	event.Subscribe(m.commands)
	event.Subscribe(m.topics)
}

func (m *Mycelium) Close() {
	event.Unsubscribe(m.handles)
	event.Unsubscribe(m.commands)
	event.Unsubscribe(m.topics)
	event.Unsubscribe(m)
}

func (m *Mycelium) OnMessage(ctx message_event.IMessageContext) {

	raw := ctx.Message()
	origin := ctx.Origin()

	// pipe message
	if message.IsPipe(raw) {

		// witness.Incoming(raw, n.registry.ID)
		// pipeMsg, ok := message.Pipe(raw, n.registry.ID)
		// if !ok {
		// 	// no parsed message to attach, so report internally instead of echoing raw text back over the wire
		// 	witness.Error(error.Malformed, error.Node, "failed to parse pipe", out.Pair("node", n.registry.ID), out.Pair("raw", raw))
		// 	continue
		// }

		// go n.bus.Pipe(pipeMsg, n)

		// witnessing starts with witness
	} else if strings.HasPrefix(raw, "witness ") { // witnessing starts with 'witness'

		msg := strings.TrimPrefix(raw, "witness ")
		witness.Node(msg, origin)

		// publishing starts with publish
	} else if strings.HasPrefix(raw, "publish") { // pub/sub starts with 'publish'

		witness.Incoming(raw, origin)
		
		raw = strings.TrimPrefix(raw, "publish ")

		topic, ok := extractTopic(raw)
		if !ok {
			// nothing to respond
			return
		}

		subs, ok := m.topics.retrieveSubs(topic, origin)
		if !ok {
			// nothing to respond
			return
		}

		for _, sub := range subs {
			broadcast := message_event.Broadcast(raw, origin, topic, sub)
			event.Dispatch().Broadcast.Notify(broadcast)
		}

	} else if strings.HasPrefix(raw, "~") { // response starts with a handle

		witness.Incoming(raw, origin)

		handle, ok := extractHandleFromResponse(raw)
		if !ok {
			// no handle
			// cannot respond
			return
		}

		destination, ok := m.handles.consumeHandle(handle)
		if !ok {
			// handle not found
			// cannot respond
			return
		}

		response := message_event.Response(raw, origin, handle, destination)
		m.pending.Receive(response)
		event.Dispatch().Response.Notify(response)

	} else { // fallback to request

		witness.Incoming(raw, origin)

		handle, ok := extractHandleFromRequest(raw)
		if !ok {
			// no handle
			// cannot respond
			return
		}

		command, ok := extractCommandFromRequest(raw)
		if !ok {
			reserr := message_event.ResponseError(error.Malformed, error.Mycelium, "bad command", handle, "n/a", origin)
			event.Dispatch().Response.Notify(reserr)
			return
		}

		ok = m.handles.mapHandle(handle, origin)
		if !ok {
			reserr := message_event.ResponseError(error.Collision, error.Mycelium, "handle collision", handle, command, origin)
			event.Dispatch().Response.Notify(reserr)
			return
		}

		node, ok := m.commands.retrieveNode(command)
		if !ok {
			reserr := message_event.ResponseError(error.Missing, error.Mycelium, "unknown command", handle, command, origin)
			event.Dispatch().Response.Notify(reserr)
			return
		}

		request := message_event.Request(raw, origin, command, node)
		event.Dispatch().Request.Notify(request)
	}
}

func (m *Mycelium) OnSynchronizedRequest(ctx sync_event.ISynchronizedRequestContext) {
	raw := ctx.Message()

	if message.IsPipe(raw) {

		res := message_event.ResponseError(error.NotApplicable, error.Mycelium, "pipe does not support synchronized requests", "n/a", "n/a", "n/a")
		ctx.Respond(res)

	} else if strings.HasPrefix(raw, "witness") {
		
		res := message_event.ResponseError(error.NotApplicable, error.Mycelium, "witness does not support synchronized requests", "n/a", "n/a", "n/a")
		ctx.Respond(res)

	} else if strings.HasPrefix(raw, "publish") {

		res := message_event.ResponseError(error.NotApplicable, error.Mycelium, "publish does not support synchronized requests", "n/a", "n/a", "n/a")
		ctx.Respond(res)

	} else if strings.HasPrefix(raw, "~") {

		res := message_event.ResponseError(error.NotApplicable, error.Mycelium, "response not supported", "n/a", "n/a", "n/a")
		ctx.Respond(res)

	} else {

		witness.Spore(raw, out.Pair("origin", ctx.Origin()))
		
		handle, ok := extractHandleFromRequest(raw)
		if !ok {
			res := message_event.ResponseError(error.Malformed, error.Mycelium, "bad handle", "n/a", "n/a", ctx.Origin())
			ctx.Respond(res)
			return
		}

		command, ok := extractCommandFromRequest(raw)
		if !ok {
			res := message_event.ResponseError(error.Malformed, error.Mycelium, "bad command", handle, "n/a", ctx.Origin())
			ctx.Respond(res)
			return
		}

		ok = m.handles.mapHandle(handle, ctx.Origin())
		if !ok {
			res := message_event.ResponseError(error.Collision, error.Mycelium, "handle collision", handle, command, ctx.Origin())
			ctx.Respond(res)
			return
		}

		node, ok := m.commands.retrieveNode(command)
		if !ok {
			res := message_event.ResponseError(error.Missing, error.Mycelium, "unknown command", handle, command, ctx.Origin())
			ctx.Respond(res)
			return
		}

		request := message_event.Request(raw, ctx.Origin(), command, node)
		ch := m.pending.Await(handle)
		event.Dispatch().Request.Notify(request)
		response, ok := m.pending.WaitFor(handle, ch) 
		if !ok {
			res := message_event.ResponseError(error.Timeout, error.Mycelium, "wait timed out", handle, command, ctx.Origin())
			ctx.Respond(res)
			return
		}
		ctx.Respond(response)
	}
}