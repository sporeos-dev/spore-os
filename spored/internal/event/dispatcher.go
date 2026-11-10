package event

import (
	"spored/internal/base"
	manifest_event "spored/internal/event/manifest"
	message_event "spored/internal/event/message"
	witness_event "spored/internal/event/witness"
)

type Dispatcher struct {
	Broadcast *ComplexEvent[message_event.IBroadcastHandler, message_event.IBroadcastContext]
	ManifestAnnouncement *SimpleEvent[manifest_event.IManifestAnnouncmentHandler, manifest_event.IManifestAnnouncementContext]
	ManifestWithdrawal *SimpleEvent[manifest_event.IManifestWithdrawalHandler, manifest_event.IManifestWithdrawalContext]
	Message *SimpleEvent[message_event.IMessageHandler, message_event.IMessageContext]
	Request *ComplexEvent[message_event.IRequestHandler, message_event.IRequestContext]
	Response *ComplexEvent[message_event.IResponseHandler, message_event.IResponseContext]
	Witness *SimpleEvent[witness_event.IWitnessHandler, witness_event.IWitnessContext]
}

func New() *Dispatcher {
	return &Dispatcher{
		Broadcast: Complex(
			func(sub message_event.IBroadcastHandler, ctx message_event.IBroadcastContext) {
				sub.OnBroadcast(ctx)
			},
			func(sub message_event.IBroadcastHandler) []string {
				return sub.Ids()
			},
			func(ctx message_event.IBroadcastContext) string {
				return ctx.Destination()
			},
		),
		ManifestAnnouncement: Simple(
			func(sub manifest_event.IManifestAnnouncmentHandler, ctx manifest_event.IManifestAnnouncementContext) {
				sub.OnManifestAnnouncement(ctx)
			},
		),
		ManifestWithdrawal: Simple(
			func(sub manifest_event.IManifestWithdrawalHandler, ctx manifest_event.IManifestWithdrawalContext) {
				sub.OnManifestWithdrawal(ctx)
			},
		),
		Message: Simple(
			func(sub message_event.IMessageHandler, ctx message_event.IMessageContext) {
				sub.OnMessage(ctx)
			},
		),
		Request: Complex(
			func(sub message_event.IRequestHandler, ctx message_event.IRequestContext) {
				sub.OnRequest(ctx)
			},
			func(sub message_event.IRequestHandler) []string {
				return sub.Api()
			},
			func(ctx message_event.IRequestContext) string {
				return ctx.Command()
			},
		),
		Response: Complex(
			func(sub message_event.IResponseHandler, ctx message_event.IResponseContext) {
				sub.OnResponse(ctx)
			},
			func(sub message_event.IResponseHandler) []string {
				return sub.Ids()
			},
			func(ctx message_event.IResponseContext) string {
				return ctx.Destination()
			},
		),
		Witness: Simple(
			func(sub witness_event.IWitnessHandler, ctx witness_event.IWitnessContext) { 
				sub.OnWitness(ctx)
			},
		),
	}
}

func (d *Dispatcher) Sub(base base.IBase) {
	d.Broadcast.Sub(base)
	d.ManifestAnnouncement.Sub(base)
	d.ManifestWithdrawal.Sub(base)
	d.Message.Sub(base)
	d.Request.Sub(base)
	d.Response.Sub(base)
	d.Witness.Sub(base)
}

func (d *Dispatcher) Unsub(base base.IBase) {
	d.Broadcast.Unsub(base)
	d.ManifestAnnouncement.Unsub(base)
	d.ManifestWithdrawal.Unsub(base)
	d.Message.Unsub(base)
	d.Request.Unsub(base)
	d.Response.Unsub(base)
	d.Witness.Unsub(base)
}