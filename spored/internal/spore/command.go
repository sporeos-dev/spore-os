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

			commands = append(commands, s.manifest.CommandIds()...)
			for _, nodeid := range s.nodes.GetNodes() {
				manifest := s.nodes.GetManifest(nodeid)
				commands = append(commands, manifest.CommandIds()...)
			}

		} else if node == s.manifest.ID {

			commands = s.manifest.CommandIds()

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

		for _, command := range s.manifest.Api {
			if commandid != command.Name {
				continue
			}

			inputs := []string{}
			if command.Inputs != nil {
				for _, input := range *command.Inputs {
					req := ""
					if input.Required {
						req = " *"
					}
					inputs = append(inputs, input.Name+" ("+input.Type+")"+req+": "+input.Description)
				}
			}

			outputs := []string{}
			if command.Outputs != nil {
				for _, output := range *command.Outputs {
					outputs = append(outputs, output.Name+" ("+output.Type+"): "+output.Description)
				}
			}

			s.respond(
				request,
				out.Array(command.Name,
					[]string{
						command.Description,
						s.manifest.ID,
					}),
				out.Array("Usage", command.Usage),
				out.Array("Inputs", inputs),
				out.Array("Outputs", outputs))
			return
		}

		for _, nodeid := range s.nodes.GetNodes() {
			
			manifest := s.nodes.GetManifest(nodeid)
			for _, command := range manifest.Api {
				if commandid != command.Name {
					continue
				}
				
				inputs := []string{}
				if command.Inputs != nil {
					for _, input := range *command.Inputs {
						req := ""
						if input.Required {
							req = " *"
						}
						inputs = append(inputs, input.Name+" ("+input.Type+")"+req+": "+input.Description)
					}
				}

				outputs := []string{}
				if command.Outputs != nil {
					for _, output := range *command.Outputs {
						outputs = append(outputs, output.Name+" ("+output.Type+"): "+output.Description)
					}
				}

				s.respond(
					request,
					out.Array(command.Name,
						[]string{
							command.Description,
							nodeid,
						}),
					out.Array("Usage", command.Usage),
					out.Array("Inputs", inputs),
					out.Array("Outputs", outputs))
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
