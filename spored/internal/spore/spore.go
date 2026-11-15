// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package spore

import (
	"fmt"
	"log/slog"
	"spored/internal/base"
	"spored/internal/cparser"
	"spored/internal/event"
	manifest_event "spored/internal/event/manifest"
	message_event "spored/internal/event/message"
	"spored/internal/manifest"
	"spored/internal/pal"
	"spored/internal/utilities/status"
	"spored/internal/witness"
)

var _ message_event.IRequestHandler = (*Spore)(nil)

type Spore struct {
	*base.Base
	manifest *manifest.Manifest
}

func New() *Spore {
	return &Spore{
		Base: base.New().SetName("spore"),
		manifest: nil,
	}
}

func (s *Spore) Open() {
	m := &manifest.Manifest{
		Status:           status.New(),
		Path:             pal.FileSporeManifest(),
		ExpectedChecksum: "n/a",
	}
	m.Status.Set(status.Verified)
	slog.Debug("Spore manifest loaded.", "manifest", m.Path)

	s.manifest = m
	m.Load()

	event.Subscribe(s)
}

func (s *Spore) Close() {
	event.Unsubscribe(s)

	man := manifest_event.New(s.manifest)
	event.Dispatch().ManifestWithdrawal.Notify(man)
}

func (s *Spore) Announce() {
	man := manifest_event.New(s.manifest)
	event.Dispatch().ManifestAnnouncement.Notify(man)
}

func(s *Spore) OnRequest(ctx message_event.IRequestContext) {

	raw := ctx.Message()
	// origin := ctx.Origin()
	cmd := ctx.Command()

	go func() {

		_, ok := cparser.ParseAndValidate(raw)
		if !ok {
			return
		}

		witness.Spore(
			fmt.Sprintf("raw: %s", raw))

		switch cmd {
		case "dev.sporeos.SPORE.node.install":
		case "dev.sporeos.SPORE.node.uninstall":
		case "dev.sporeos.SPORE.node.reinstall":

		case "dev.sporeos.SPORE.node.spawn":
		case "dev.sporeos.SPORE.node.kill":

		case "dev.sporeos.SPORE.topic.subscribe":
		case "dev.sporeos.SPORE.topic.unsubscribe":
			
		case "dev.sporeos.SPORE.permission.request":
		case "dev.sporeos.SPORE.permission.grant":
		case "dev.sporeos.SPORE.permission.revoke":

		case "dev.sporeos.SPORE.state.persist":
		case "dev.sporeos.SPORE.state.retrieve":

		default:
			
		}
	}()
}

func (s *Spore) Api() []string {
	api := []string{}
	for _, el := range s.manifest.Api {
		api = append(api, el.Name)
	}
	return api
}
