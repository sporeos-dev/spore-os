package spore

import (
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
)

func (s *Spore) errorList(request icast) *error.Error {

	node := request.ArgIf("node", "all")
	
	go func() {

		errors := make([]string, 0)

		if node == "all" {

			for _, nodeid := range s.nodes.GetNodes() {
				manifest := s.nodes.GetManifest(nodeid)
				errors = append(errors, manifest.ErrorIds()...)
			}

		} else {

			manifest := s.nodes.GetManifest(node)
			errors = manifest.ErrorIds()

		}

		s.respond(request, out.NewArray("errors", errors))
	}()

	return nil
}

func (s *Spore) errorHelp(request icast) *error.Error {

	errid, err := request.Arg("error")
	if err == nil {
		return err
	}

	go func() {

		for _, nodeid := range s.nodes.GetNodes() {
			
			manifest := s.nodes.GetManifest(nodeid)
			for _, err := range manifest.Errors {
				if errid != err.Name {
					continue
				}
				
				response := map[string]any {
					"id": err.Name,
					"name": err.Name,
					"description": err.Description,
				}
				
				s.respond(request, out.NewObject("errorhelp", response))
				return
			}
		}
		
		s.respondError(request, error.New(error.RouteNotFound, "error not found"))
	}()

	return nil
}
