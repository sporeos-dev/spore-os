package spore

import (
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
)

func (s *Spore) commandList(request icast) *error.Error {

	node := request.ArgIf("node", "all")
	
	go func() {

		commands := make([]string, 0)
		
		if node == "all" {

			for _, nodeid := range s.nodes.GetNodes() {
				manifest := s.nodes.GetManifest(nodeid)
				commands = append(commands, manifest.CommandIds()...)
			}

		} else {

			manifest := s.nodes.GetManifest(node)
			if manifest == nil {
				s.respondError(request, error.New(error.RouteNotFound, "node not found"))
				return
			}
			commands = manifest.CommandIds()

		}

		s.respond(request, out.NewArray("commands", commands))

	}()

	return nil
}

func (s *Spore) commandHelp(request icast) *error.Error {

	commandid, err := request.Arg("command")
	if err != nil {
		return err
	}

	go func() {

		for _, nodeid := range s.nodes.GetNodes() {
			
			manifest := s.nodes.GetManifest(nodeid)
			for _, command := range manifest.Api {
				if commandid != command.Name {
					continue
				}
				
				response := map[string]any {
					"id": command.Name,
					"name": command.Name,
					"description": command.Description,
					"node": nodeid,
					"inputs": command.Inputs,
					"outputs": command.Outputs,
				}

				s.respond(request, out.NewObject("commandhelp", response))
				return
			}
		}

		s.respondError(request, error.New(error.RouteNotFound, "command not found"))
	}()

	return nil
}
