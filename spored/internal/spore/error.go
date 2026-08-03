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
				s.bus.Response(
					error.New(
						error.Missing,
						error.Spore,
						"spore manifest not loaded").
						WithMessage(request))
				return
			}
			errors = manifest.ErrorIds()

		}

		s.bus.Response(message.Spore(request, out.Array("errors", errors)))
	}()

	return nil
}

func (s *Spore) errorHelp(request message.Message) *error.Error {

	errid, ok := request.Arg("code")
	if !ok {
		return error.MissingArg("code", error.Spore).WithMessage(request)
	}

	go func() {

		for _, err := range s.manifest.Errors {
			if errid != err.Name {
				continue
			}
			s.bus.Response(
				message.Spore(
					request,
					out.Array(err.Name,
						[]string{
							"Description: " + err.Description,
						})))
			return
		}

		for _, nodeid := range s.nodes.GetNodes() {
			
			manifest := s.nodes.GetManifest(nodeid)
			for _, err := range manifest.Errors {
				if errid != err.Name {
					continue
				}
				
				s.bus.Response(
					message.Spore(
						request,
						out.Array(err.Name,
							[]string{
								"Description: " + err.Description,
							})))
				return
			}
		}
		
		s.bus.Response(
			error.New(
				error.Missing,
				error.Spore,
				"error not found",
				out.Pair("error", errid)).
				WithMessage(request))
	}()

	return nil
}
