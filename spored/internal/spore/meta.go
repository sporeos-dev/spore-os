package spore

import (
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
)

func (s *Spore) help(request icast) *error.Error {

	go func() {
		response := map[string]any {
			"id": "SPORE.help",
			"name": "Spore OS Help",
			"description": "Getting Started...",
			
			"contact": map[string]string {
				"github": "github.com/sporeos-dev",
				"website": "sporeos.dev",
			},

			"getting_started": []string {
				"SPORE.info",
				"SPORE.node.list",
				"SPORE.node.help node=node",
				"SPORE.command.list",
				"SPORE.command.help command=command",
			},
			
			"version": map[string]string {
				"schema": s.manifest.Schema,
				"version": s.manifest.Version,
			},
		}

		s.respond(request, out.NewObject("help", response))
	}()
	
	return nil
}

func (s *Spore) info(request icast) *error.Error {

	go func() {
		response := map[string]any {
			"id": s.manifest.ID,
			"name": s.manifest.Name,
			"description": s.manifest.Description,
			"api": s.manifest.CommandIds(),
			"topics": s.manifest.TopicIds(),
		}

		s.respond(request, out.NewObject("info", response))
	}()

	return nil
}

func (s *Spore) state(request icast) *error.Error {

	go func() {
		s.respondError(request, error.New(error.RouteNotImplemented, "not yet implemented"))
	}()

	return nil
}
