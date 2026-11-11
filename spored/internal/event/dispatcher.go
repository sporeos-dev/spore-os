package event

import (
	"spored/internal/base"
	app_event "spored/internal/event/app"
	manifest_event "spored/internal/event/manifest"
	message_event "spored/internal/event/message"
	node_event "spored/internal/event/node"
	registry_event "spored/internal/event/registry"
	sync_event "spored/internal/event/sync"
	witness_event "spored/internal/event/witness"
)

type Dispatcher struct {
	AppEntry *EmptyEvent[app_event.IAppEntryHandler]
	AppExit *EmptyEvent[app_event.IAppExitHandler]
	Broadcast *ComplexEvent[message_event.IBroadcastHandler, message_event.IBroadcastContext]
	ManifestAnnouncement *SimpleEvent[manifest_event.IManifestAnnouncmentHandler, manifest_event.IManifestAnnouncementContext]
	ManifestWithdrawal *SimpleEvent[manifest_event.IManifestWithdrawalHandler, manifest_event.IManifestWithdrawalContext]
	Message *SimpleEvent[message_event.IMessageHandler, message_event.IMessageContext]
	NodeConnection *IdentityEvent[node_event.INodeConnectionHandler]
	NodeDisconnection *IdentityEvent[node_event.INodeDisconnectionHandler]
	RegistryAnnouncement *SimpleEvent[registry_event.IRegistryAnnouncementHandler, registry_event.IRegistryContext]
	RegistryWithdrawal *SimpleEvent[registry_event.IRegistryWithdrawalHandler, registry_event.IRegistryContext]
	Request *ComplexEvent[message_event.IRequestHandler, message_event.IRequestContext]
	Response *ComplexEvent[message_event.IResponseHandler, message_event.IResponseContext]
	SynchronizedRequest *SimpleEvent[sync_event.ISynchronizedRequestHandler, sync_event.ISynchronizedRequestContext]
	Witness *SimpleEvent[witness_event.IWitnessHandler, witness_event.IWitnessContext]
}

func New() *Dispatcher {
	return &Dispatcher{
		AppEntry: Empty(
			func(sub app_event.IAppEntryHandler) {
				sub.OnAppEntry()
			},
		),
		AppExit: Empty(
			func(sub app_event.IAppExitHandler) {
				sub.OnAppExit()
			},
		),
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
		NodeConnection: Identity(
			func(sub node_event.INodeConnectionHandler, id string) {
				sub.OnNodeConnection(id)
			},
		),
		NodeDisconnection: Identity(
			func(sub node_event.INodeDisconnectionHandler, id string) {
				sub.OnNodeDisconnection(id)
			},
		),
		RegistryAnnouncement: Simple(
			func(sub registry_event.IRegistryAnnouncementHandler, ctx registry_event.IRegistryContext) {
				sub.OnRegistryAnnouncement(ctx)
			},
		),
		RegistryWithdrawal: Simple(
			func(sub registry_event.IRegistryWithdrawalHandler, ctx registry_event.IRegistryContext) {
				sub.OnRegistryWithdrawal(ctx)
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
		SynchronizedRequest: Simple(
			func(sub sync_event.ISynchronizedRequestHandler, ctx sync_event.ISynchronizedRequestContext) {
				sub.OnSynchronizedRequest(ctx)
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
	d.AppEntry.Sub(base)
	d.AppExit.Sub(base)
	d.Broadcast.Sub(base)
	d.ManifestAnnouncement.Sub(base)
	d.ManifestWithdrawal.Sub(base)
	d.Message.Sub(base)
	d.NodeConnection.Sub(base)
	d.NodeDisconnection.Sub(base)
	d.RegistryAnnouncement.Sub(base)
	d.RegistryWithdrawal.Sub(base)
	d.Request.Sub(base)
	d.Response.Sub(base)
	d.SynchronizedRequest.Sub(base)
	d.Witness.Sub(base)
}

func (d *Dispatcher) Unsub(base base.IBase) {
	d.AppEntry.Unsub(base)
	d.AppExit.Unsub(base)
	d.Broadcast.Unsub(base)
	d.ManifestAnnouncement.Unsub(base)
	d.ManifestWithdrawal.Unsub(base)
	d.Message.Unsub(base)
	d.NodeConnection.Unsub(base)
	d.NodeDisconnection.Unsub(base)
	d.RegistryAnnouncement.Unsub(base)
	d.RegistryWithdrawal.Unsub(base)
	d.Request.Unsub(base)
	d.Response.Unsub(base)
	d.SynchronizedRequest.Unsub(base)
	d.Witness.Unsub(base)
}