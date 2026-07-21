package spore

import (
	"spored/internal/message"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
)

func (s *Spore) commandList(request message.Message) *error.Error {

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
				err := error.New(
					error.Missing,
					error.Spore,
					"spore manifest not loaded",
					out.Pair("command", "SPORE.command.list"))
				s.respondError(request, err)
				return
			}
			commands = manifest.CommandIds()

		}

		s.respond(request, out.Array("commands", commands))

	}()

	return nil
}

func (s *Spore) commandHelp(request message.Message) *error.Error {

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

				s.respond(request, out.Object("commandhelp", response))
				return
			}
		}

		err := error.New(
			error.Missing,
			error.Spore,
			"command not found",
			out.Pair("command", "SPORE.command.help"))
		s.respondError(request, err)
	}()

	return nil
}
