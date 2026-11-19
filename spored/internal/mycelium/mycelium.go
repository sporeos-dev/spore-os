package mycelium

import (
	"spored/internal/base"
	"spored/internal/event"
	message_event "spored/internal/event/message"
	sync_event "spored/internal/event/sync"
	"spored/internal/permission"
	"spored/internal/utilities/await"
	"spored/internal/utilities/error"
	"spored/internal/utilities/message"
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
	allowance *permission.Allowance
}

func New() *Mycelium {
	return &Mycelium{
		Base:     base.New().SetName("mycelium"),
		handles:  newHandleMap(),
		commands: newCommandMap(),
		topics: newTopics(),
		pending: await.New(error.Mycelium).WithTimeout(defaultTimeout),
		allowance: permission.New(),
	}
}

func (m *Mycelium) Open() {
	event.Subscribe(m)
	event.Subscribe(m.handles)
	event.Subscribe(m.commands)
	event.Subscribe(m.topics)
	event.Subscribe(m.allowance)
}

func (m *Mycelium) Close() {
	event.Unsubscribe(m.handles)
	event.Unsubscribe(m.commands)
	event.Unsubscribe(m.topics)
	event.Unsubscribe(m.allowance)
	event.Unsubscribe(m)
}

func (m *Mycelium) OnMessage(ctx message_event.IMessageContext) {

	raw := ctx.Message()
	origin := ctx.Origin()

	// allows nodes to set themselves up in case they haven't yet
	event.Dispatch().BeforeMessage.Notify()

	// pipe message
	if len(splitPipe(raw)) > 1 {
		witness.Incoming(raw, origin)
		m.startPipe(raw, origin)

		// witnessing starts with witness
	} else if strings.HasPrefix(raw, "witness ") { // witnessing starts with 'witness'

		msg := strings.TrimPrefix(raw, "witness ")
		witness.Node(msg, origin)

		// publishing starts with publish
	} else if strings.HasPrefix(raw, "publish") { // pub/sub starts with 'publish'

		witness.Incoming(raw, origin)
		if !ensureNoCast(raw) {
			return
		}
		
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
		if !ensureNoCapture(raw) {
			return
		}

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
		if m.pending.Has(handle) {
			m.pending.Receive(response)
		} else {
			event.Dispatch().Response.Notify(response)
		}

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

		if !ensureNoCast(raw) {
			reserr := message_event.ResponseError(error.ReservedLanguage, error.Mycelium, "cast is hub-reserved", handle, command, origin)
			event.Dispatch().Response.Notify(reserr)
			return
		}

		ok = m.handles.mapHandle(handle, origin)
		if !ok {
			reserr := message_event.ResponseError(error.Collision, error.Mycelium, "handle collision", handle, command, origin)
			event.Dispatch().Response.Notify(reserr)
			return
		}

		node, fqName, ok := m.commands.retrieveNodeAndFqName(command)
		if !ok {
			reserr := message_event.ResponseError(error.Missing, error.Mycelium, "unknown command", handle, command, origin)
			event.Dispatch().Response.Notify(reserr)
			m.handles.deleteHandle(handle)
			return
		}

		can := m.allowance.Can(origin, fqName, raw)
		if !can {
			reserr := message_event.ResponseError(error.NotPermitted, error.Mycelium, "not allowed", handle, command, origin)
			event.Dispatch().Response.Notify(reserr)
			m.handles.deleteHandle(handle)
			return
		}

		raw = message.RemoveFlag(raw, "request.permission")
		request := message_event.Request(raw, origin, fqName, node, handle)
		event.Dispatch().Request.Notify(request)
	}
}

func (m *Mycelium) OnSynchronizedRequest(ctx sync_event.ISynchronizedRequestContext) {
	raw := ctx.Message()

	if len(splitPipe(raw)) > 1 {

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

		if !ensureNoCast(raw) {
			res := message_event.ResponseError(error.ReservedLanguage, error.Mycelium, "cast is hub-reserved", handle, command, ctx.Origin())
			ctx.Respond(res)
			return
		}

		ok = m.handles.mapHandle(handle, ctx.Origin())
		if !ok {
			res := message_event.ResponseError(error.Collision, error.Mycelium, "handle collision", handle, command, ctx.Origin())
			ctx.Respond(res)
			return
		}

		node, fqCmd, ok := m.commands.retrieveNodeAndFqName(command)
		if !ok {
			m.handles.deleteHandle(handle)
			res := message_event.ResponseError(error.Missing, error.Mycelium, "unknown command", handle, command, ctx.Origin())
			ctx.Respond(res)
			return
		}

		origin := ctx.Origin()
		if actor, caller, delegated := strings.Cut(origin, ":"); delegated {
			origin = caller
			if actor == "dev.sporeos.SPORE" && fqCmd == "dev.sporeos.SPORE.permission.request" {
				origin = actor
			}
		}
		can := m.allowance.Can(origin, fqCmd, raw)
		if !can {
			m.handles.deleteHandle(handle)
			res := message_event.ResponseError(error.NotPermitted, error.Mycelium, "not allowed", handle, command, ctx.Origin())
			ctx.Respond(res)
			return
		}

		raw = message.RemoveFlag(raw, "request.permission")
		request := message_event.Request(raw, ctx.Origin(), fqCmd, node, handle)
		ch := m.pending.Await(handle)
		var timeout *time.Duration
		timeoutModule := error.Mycelium
		if timed, ok := ctx.(interface{ TimeoutOverride() *time.Duration }); ok {
			timeout = timed.TimeoutOverride()
			timeoutModule = error.Pipe
			go event.Dispatch().Request.Notify(request)
		} else {
			event.Dispatch().Request.Notify(request)
		}
		response, ok := m.pending.WaitForTimeout(handle, ch, timeout)
		if !ok {
			m.handles.deleteHandle(handle)
			res := message_event.ResponseError(error.Timeout, timeoutModule, "wait timed out", handle, command, ctx.Origin())
			ctx.Respond(res)
			return
		}

		ctx.Respond(response)
	}
}