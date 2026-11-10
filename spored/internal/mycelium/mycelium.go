package mycelium

import (
	"spored/internal/base"
	"spored/internal/event"
	message_event "spored/internal/event/message"
	"spored/internal/message"
	"spored/internal/utilities/error"
	"spored/internal/witness"
	"strings"
)

type Mycelium struct {
	*base.Base
	handles  *handleMap
	commands *commandMap
	topics   *topics
}

func New() *Mycelium {
	return &Mycelium{
		Base:     base.New().SetName("mycelium"),
		// request/response
		handles:  newHandleMap(),
		commands: newCommandMap(),
		// broadcast
		topics: newTopics(),
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

