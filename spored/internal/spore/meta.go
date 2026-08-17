package spore

import (
	"spored/internal/iface"
	"spored/internal/message"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
)

func (s *Spore) help(request iface.Message) *error.Error {

	go func() {
		s.bus.Response(
			message.Spore(
				request, 
				out.Array("SPORE.help", 
					[]string{
						"Spore OS Help",
						"Schema: " + s.manifest.Schema,
						"Version: " + s.manifest.Version,
						"sporeos.dev",
						"github.com/sporeos-dev",
					}),
				out.Array("Getting-started-commands...", 
					[]string{
						"SPORE.info",
						"SPORE.node.list",
						"SPORE.node.help node=node",
						"SPORE.command.list",
						"SPORE.command.help command=command",
					})))
	}()
	
	return nil
}

func (s *Spore) info(request iface.Message) *error.Error {

	go func() {
		s.bus.Response(
			message.Spore(
				request,
				out.Array("SPORE.info",
					[]string{
						s.manifest.ID,
						s.manifest.Name,
						s.manifest.Description,
						"Schema: " + s.manifest.Schema,
						"Version: " + s.manifest.Version,
					}),
				out.Array("API",
					s.manifest.CommandIds()),
				out.Array("Topics",
					s.manifest.TopicIds())))
	}()

	return nil
}

func (s *Spore) state(request iface.Message) *error.Error {

	go func() {
		s.bus.Response(
			error.New(
				error.NotImplemented,
				error.Spore,
				"spore state not yet implemented").
				WithMessage(request))
	}()

	return nil
}
