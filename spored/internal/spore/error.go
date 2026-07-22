package spore

import (
	"spored/internal/message"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
)

func (s *Spore) errorList(request message.Message) *error.Error {

	node := request.ArgIf("node", "all")
	
	go func() {

		errors := make([]string, 0)

		if node == "all" {

			errors = append(errors, s.manifest.ErrorIds()...)
			for _, nodeid := range s.nodes.GetNodes() {
				manifest := s.nodes.GetManifest(nodeid)
				errors = append(errors, manifest.ErrorIds()...)
			}

		} else if node == s.manifest.ID {

			errors = s.manifest.ErrorIds()

		} else {

			manifest := s.nodes.GetManifest(node)
			if manifest == nil {
				err := error.New(
					error.Missing,
					error.Spore,
					"spore manifest not loaded",
					out.Pair("node", node))
				s.respondError(request, err)
				return
			}
			errors = manifest.ErrorIds()

		}

		s.respond(request, out.Array("errors", errors))
	}()

	return nil
}

func (s *Spore) errorHelp(request message.Message) *error.Error {

	errid, err := request.Arg("error")
	if err != nil {
		return err
	}

	go func() {

		for _, err := range s.manifest.Errors {
			if errid != err.Name {
				continue
			}
			s.respond(
				request,
				out.Array(err.Name,
					[]string{
						"Description: " + err.Description,
					}))
			return
		}

		for _, nodeid := range s.nodes.GetNodes() {
			
			manifest := s.nodes.GetManifest(nodeid)
			for _, err := range manifest.Errors {
				if errid != err.Name {
					continue
				}
				
				s.respond(
					request,
					out.Array(err.Name,
						[]string{
							"Description: " + err.Description,
						}))
				return
			}
		}
		
		err := error.New(
			error.Missing,
			error.Spore,
			"error not found",
			out.Pair("error", errid))
		s.respondError(request, err)
	}()

	return nil
}
