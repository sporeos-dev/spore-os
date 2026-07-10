package spore

import (
	"encoding/json"
	"errors"
	"spored/internal/manifest"
	"spored/internal/message"
	"strings"
)

func (s *Spore) commandList(msg *message.Spore) (string, error) {

	nodeid := msg.GetArgumentIf("node", "n/a")

	var commands []string
	if nodeid == "dev.sporeos.SPORE" {
		// SPORE hub commands are not in the router map; read directly from manifest.
		if s.manifest == nil {
			return "", errors.New("SPORE manifest not loaded")
		}
		for _, command := range s.manifest.Api {
			commands = append(commands, command.Name)
		}
	} else {
		commands = s.router.ListCommands(nodeid)
		if nodeid == "n/a" && s.manifest != nil {
			for _, command := range s.manifest.Api {
				commands = append(commands, command.Name)
			}
		}
	}

	// list commands
	println()
	println("Commands:")
	for _, command := range commands {
		println("  -", command)
	}
	println()

	// serialize and return
	serialized, err := json.Marshal(commands)
	if err != nil {
		return "", err
	}

	return s.returnCapture(msg, map[string]string{
		"commands": string(serialized),
	}, []string{}), nil
}

func (s *Spore) commandHelp(msg *message.Spore) (string, error) {

	// get command definition from manifest
	commandid, err := msg.GetArgument("command")
	if err != nil {
		return "", err
	}

	// Check if this is a SPORE command and handle specially
	var command manifest.Command
	var nodeManifest *manifest.Manifest

	if strings.HasPrefix(commandid, "SPORE.") {
		// SPORE command: look up in SPORE manifest directly
		if s.manifest == nil {
			return "", errors.New("SPORE manifest not loaded")
		}
		nodeManifest = s.manifest
		success := false
		for _, cmd := range nodeManifest.Api {
			if cmd.Name == commandid {
				command = cmd
				success = true
				break
			}
		}
		if !success {
			return "", errors.New("SPORE command not found: " + commandid)
		}
	} else {
		// Regular command: use router to find the node
		nodeid, err := s.router.GetRoute(commandid)
		if err != nil {
			return "", err
		}

		node, err := s.hub.GetNode(nodeid)
		if err != nil {
			return "", err
		}

		nodeManifest = node.GetManifest()
		success := false
		for _, cmd := range nodeManifest.Api {
			if cmd.Name == commandid {
				command = cmd
				success = true
				break
			}
		}
		if !success {
			return "", errors.New("command not found")
		}
	}

	// show command help
	println()
	println("  " + command.Name)
	println("  " + command.Description)
	println("  ----------")
	
	println("  usage:")
	usage := []string{}
	for _, el := range command.Usage {
		println("  -", el)
		usage = append(usage, el)
	}

	inputs := []string{}
	if command.Inputs != nil {
		for _, el := range *command.Inputs {
			println("  -", el.Name, "(" + el.Type + ")", el.Description)
			inputs = append(inputs, el.Name + ", " + el.Type + ", " + el.Description)
		}
	}

	outputs := []string{}
	if command.Outputs != nil {
		for _, el := range *command.Outputs {
			println("  -", el.Name, "(" + el.Type + ")", el.Description)
			outputs = append(outputs, el.Name + ", " + el.Type + ", " + el.Description)
		}
	}

	notes := []string{}
	if command.Notes != nil {
		for _, el := range *command.Notes {
			println("  -", el)
			notes = append(notes, el)
		}
	}
	println()

	// serialize and return
	result := map[string]interface{}{
		"name": command.Name,
		"description": command.Description,
		"usage": command.Usage,
		"inputs": inputs,
		"outputs": outputs,
		"notes": notes,
	}

	serialized, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return s.returnCapture(msg, map[string]string{
		"commandinfo": string(serialized),
	}, []string{}), nil
}