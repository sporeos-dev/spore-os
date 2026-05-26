package spore

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"spored/internal/interfaces"
	"spored/internal/manifest"
	"spored/internal/message"
	"strings"
)

type Spore struct {
	hub interfaces.Hub
	registry interfaces.Registry
	router interfaces.Router
	witness interfaces.Witness

	manifest *manifest.Manifest
}

func (s *Spore) Open(hub interfaces.Hub, registry interfaces.Registry, router interfaces.Router, witness interfaces.Witness) {
	s.hub = hub
	s.registry = registry
	s.router = router
	s.witness = witness

	manifestPath := hubManifestPath()
	if _, statErr := os.Stat(manifestPath); os.IsNotExist(statErr) {
		// System path not found — fall back to next to binary (dev / direct run).
		if exePath, err := os.Executable(); err == nil {
			manifestPath = filepath.Join(filepath.Dir(exePath), "spored.manifest.spore.yaml")
		}
	}
	m, err := manifest.LoadManifest(manifestPath)
	if err != nil {
		slog.Error("Unable to load SPORE manifest", "error", err)
		s.witness.Spore(message.SporeEvent("error", "Manifest load failed", fmt.Sprintf(`error="%s"`, err.Error())))
		return
	}
	s.manifest = m
}

func (s *Spore) Command(incoming message.Message) (message.Message, error) {
	
	sporeMessage, ok := incoming.(*message.Spore)
	if !ok {
		s.witness.Spore(message.SporeEvent("warn", "Command rejected", "type=non-spore"))
		return nil, errors.New("non-spore command cannot be run by spore")
	}

	var res string
	var err error

	switch sporeMessage.Command() {
		case "SPORE.help":				res, err = s.help(sporeMessage)
		case "SPORE.node.install": 		res, err = s.nodeInstall(sporeMessage)
		case "SPORE.node.uninstall": 	res, err = s.nodeUninstall(sporeMessage)
		case "SPORE.node.spawn": 		res, err = s.nodeSpawn(sporeMessage)
		case "SPORE.node.kill": 		res, err = s.nodeKill(sporeMessage)
		case "SPORE.node.list": 		res, err = s.nodeList(sporeMessage)
		case "SPORE.node.help":			res, err = s.nodeHelp(sporeMessage)
		case "SPORE.command.list":		res, err = s.commandList(sporeMessage)
		case "SPORE.command.help":		res, err = s.commandHelp(sporeMessage)
		case "SPORE.error.list":		res, err = s.errorList(sporeMessage)
		case "SPORE.error.help":		res, err = s.errorHelp(sporeMessage)

		default: 
			err = errors.New("Spore command not handled")
	}

	if err != nil {
		s.witness.Spore(message.SporeEvent("error", "Command failed", "command="+sporeMessage.Command(), fmt.Sprintf(`error="%s"`, err.Error())))
		return nil, err
	}

	outgoing, err := message.Parse(res, "dev.sporeos.SPORE")
	if err != nil {
		s.witness.Spore(message.SporeEvent("error", "Outgoing parse failed", "command="+sporeMessage.Command(), fmt.Sprintf(`error="%s"`, err.Error())))
		return nil, err
	}
	mid := incoming.MessageId()
	outgoing.SetMessageId(mid)
	outgoing.SetDestination(sporeMessage.Cast())

	return outgoing, nil
}

func (s *Spore) help(msg *message.Spore) (string, error) {

	if s.manifest == nil {
		return "", errors.New("SPORE manifest not loaded")
	}

	// from SPORE
	manifest := s.manifest

	// show node help
	println()
	println("  " + manifest.Name + " // " + manifest.ID)
	println("  " + manifest.Description)
	println("  Version: " + manifest.Version + " // Schema: " + manifest.Schema)
	println("  " + manifest.App)
	println("  ----------")
	
	println("  Api:")
	api := []string{}
	for _, command := range manifest.Api {
		println("  -", command.Name)
		api = append(api, command.Name)
	}
	println()

	// serialize and return
	result := map[string]interface{}{
		"id": manifest.ID,
		"name": manifest.Name,
		"description": manifest.Description,
		"version": manifest.Version,
		"schema": manifest.Schema,
		"api": api,
	}

	serialized, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return s.returnCapture(msg, map[string]string{
		"nodeinfo": string(serialized),
	}, []string{}), nil
}

func (s *Spore) nodeInstall(msg *message.Spore) (string, error) {
	path, err := msg.GetArgument("path")
	if err != nil {
		return "", err
	}

	err = s.registry.Add(path)
	if err != nil {
		return "", err
	}

	// Load the node into memory immediately so it's usable without a restart.
	if err = s.hub.AddNode(path); err != nil {
		// Roll back the registry entry so we don't leave a broken install on disk.
		s.registry.Remove(path)
		return "", err
	}
	//fmt.Sprintf(`~%s:SPORE.unknown error code=MessageMalformed what="%s"`, handle, err.Error())
	s.witness.Spore(message.SporeEvent("info", "Node installed", "path="+path))
	return s.returnCapture(msg, map[string]string{}, []string{}), nil
}

func (s *Spore) nodeUninstall(msg *message.Spore) (string, error) {
	nodeid := msg.GetArgumentIf("node", "n/a")
	path := msg.GetArgumentIf("path", "n/a")

	if nodeid == "n/a" && path == "n/a" {
		return "", errors.New("missing either 'node' or 'path'")
	} else if nodeid != "n/a" && path != "n/a" {
		return "", errors.New("only one of 'node', 'path' can be used")
	} else if nodeid != "n/a" {
		node, err := s.hub.GetNode(nodeid)
		if err != nil {
			return "", err
		}
		path = node.GetManifest().Path
	}

	// Remove from the persistent registry first.
	err := s.registry.Remove(path)
	if err != nil {
		return "", err
	}

	// Resolve node ID from path if we don't have it already.
	if nodeid == "n/a" {
		for _, id := range s.hub.ListNodes() {
			n, err := s.hub.GetNode(id)
			if err != nil {
				continue
			}
			if n.GetManifest().Path == path {
				nodeid = id
				break
			}
		}
	}

	// Remove from memory (best-effort — node may never have loaded).
	if nodeid != "n/a" {
		s.hub.RemoveNode(nodeid)
	}

	s.witness.Spore(message.SporeEvent("info", "Node uninstalled", "node="+nodeid))
	return s.returnCapture(msg, map[string]string{}, []string{}), nil
}

func (s *Spore) nodeSpawn(msg *message.Spore) (string, error) {
	nodeid, err := msg.GetArgument("node")
	if err != nil {
		return "", err
	}

	if err := s.hub.SpawnNode(nodeid); err != nil {
		return "", err
	}

	s.witness.Spore(message.SporeEvent("info", "Node spawned", "node="+nodeid))
	return s.returnCapture(msg, map[string]string{}, []string{}), nil
}

func (s *Spore) nodeKill(msg *message.Spore) (string, error) {
	nodeid, err := msg.GetArgument("node")
	if err != nil {
		return "", err
	}

	if err := s.hub.KillNode(nodeid); err != nil {
		return "", err
	}

	s.witness.Spore(message.SporeEvent("info", "Node killed", "node="+nodeid))
	return s.returnCapture(msg, map[string]string{}, []string{}), nil
}

func (s *Spore) nodeList(msg *message.Spore) (string, error) {

	withConnected := msg.HasFlag("connected")

	ids := s.hub.ListNodes()

	// Build the entries: plain IDs, or with a "connected" flag appended when live.
	entries := make([]string, 0, len(ids)+1)
	for _, id := range ids {
		entry := id
		if withConnected {
			node, err := s.hub.GetNode(id)
			if err == nil && node.IsConnected() {
				entry += " connected"
			}
		}
		entries = append(entries, entry)
	}

	// SPORE itself is always connected.
	sporeEntry := "dev.sporeos.SPORE"
	if withConnected {
		sporeEntry += " connected"
	}
	entries = append(entries, sporeEntry)

	// list nodes
	println()
	println("Nodes:")
	for _, entry := range entries {
		println("  -", entry)
	}
	println()

	// serialize and return
	serialized, err := json.Marshal(entries)
	if err != nil {
		return "", err
	}

	return s.returnCapture(msg, map[string]string{
		"nodes": string(serialized),
	}, []string{}), nil
}

func (s *Spore) nodeHelp(msg *message.Spore) (string, error) {

	nodeid, err := msg.GetArgument("node")
	if err != nil {
		return "", err
	}

	// get manifest
	var manifest *manifest.Manifest

	// from SPORE
	if nodeid == "dev.sporeos.SPORE" {

		if s.manifest == nil {
			return "", errors.New("SPORE manifest not loaded")
		}
		manifest = s.manifest

	// from other
	} else {

		node, err := s.hub.GetNode(nodeid)
		if err != nil {
			return "", err
		}
		manifest = node.GetManifest()
	}

	// show node help
	println()
	println("  " + manifest.Name + " // " + manifest.ID)
	println("  " + manifest.Description)
	println("  Version: " + manifest.Version + " // Schema: " + manifest.Schema)
	println("  " + manifest.App)
	println("  Path: " + manifest.Path)
	println("  ----------")
	
	println("  Api:")
	api := []string{}
	for _, command := range manifest.Api {
		println("  -", command.Name)
		api = append(api, command.Name)
	}
	println()

	// serialize and return
	result := map[string]interface{}{
		"id": manifest.ID,
		"name": manifest.Name,
		"description": manifest.Description,
		"version": manifest.Version,
		"schema": manifest.Schema,
		"app": manifest.App,
		"path": manifest.Path,
		"api": api,
	}

	serialized, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return s.returnCapture(msg, map[string]string{
		"nodeinfo": string(serialized),
	}, []string{}), nil
}

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

func (s *Spore) errorList(msg *message.Spore) (string, error) {
	nodeid := msg.GetArgumentIf("node", "n/a")

	var errCodes []string

	switch nodeid {
	case "dev.sporeos.SPORE":
		// Hub node — standard errors only.
		for _, info := range message.StandardErrors() {
			errCodes = append(errCodes, string(info.Code))
		}
	case "n/a":
		// No node — standard errors plus every registered node's custom errors.
		for _, info := range message.StandardErrors() {
			errCodes = append(errCodes, string(info.Code))
		}
		for _, nid := range s.hub.ListNodes() {
			node, err := s.hub.GetNode(nid)
			if err != nil {
				continue
			}
			for _, e := range node.GetManifest().Errors {
				errCodes = append(errCodes, e.Name)
			}
		}
	default:
		// Specific node — custom errors for that node only.
		node, err := s.hub.GetNode(nodeid)
		if err != nil {
			return "", err
		}
		for _, e := range node.GetManifest().Errors {
			errCodes = append(errCodes, e.Name)
		}
	}

	println()
	println("Errors:")
	for _, e := range errCodes {
		println(" -", e)
	}
	println()

	serialized, err := json.Marshal(errCodes)
	if err != nil {
		return "", err
	}

	return s.returnCapture(msg, map[string]string{
		"errors": string(serialized),
	}, []string{}), nil
}

func (s *Spore) errorHelp(msg *message.Spore) (string, error) {
	name, err := msg.GetArgument("error")
	if err != nil {
		return "", err
	}

	// Check standard errors first.
	for _, info := range message.StandardErrors() {
		if string(info.Code) == name {
			println()
			println(" "+string(info.Code))
			println(" "+info.Description)
			if len(info.Examples) > 0 {
				println(" Examples:")
				for _, ex := range info.Examples {
					println("   -", ex)
				}
			}
			println()

			result := map[string]interface{}{
				"name":        string(info.Code),
				"description": info.Description,
				"examples":    info.Examples,
			}
			serialized, err := json.Marshal(result)
			if err != nil {
				return "", err
			}
			return s.returnCapture(msg, map[string]string{
				"errorinfo": string(serialized),
			}, []string{}), nil
		}
	}

	// Not a standard code — search manifest-declared errors in registered nodes.
	for _, nodeid := range s.hub.ListNodes() {
		node, err := s.hub.GetNode(nodeid)
		if err != nil {
			continue
		}
		for _, e := range node.GetManifest().Errors {
			if e.Name == name {
				println()
				println(" "+e.Name)
				println(" "+e.Description)
				if len(e.Examples) > 0 {
					println(" Examples:")
					for _, ex := range e.Examples {
						println("   -", ex)
					}
				}
				println()

				result := map[string]interface{}{
					"name":        e.Name,
					"description": e.Description,
					"examples":    e.Examples,
				}
				serialized, err := json.Marshal(result)
				if err != nil {
					return "", err
				}
				return s.returnCapture(msg, map[string]string{
					"errorinfo": string(serialized),
				}, []string{}), nil
			}
		}
	}

	return "", errors.New("error code not found: " + name)
}

func (s *Spore) returnCapture(msg *message.Spore, args map[string]string, flags []string) string {
	parts := []string{ fmt.Sprintf("~%s:%s ok capture=dev.sporeos.SPORE", msg.Handle(), msg.Command()) }
	parts = append(parts, flags...)
	for k, v := range args {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, " ")
}

// hubManifestPath returns the system-level path for the hub's own manifest.
// Mirrors the platform logic in registry.RegistryPath().
func hubManifestPath() string {
	switch runtime.GOOS {
	case "darwin":
		return "/Library/Application Support/spore-os/hub/spored.manifest.spore.yaml"
	case "linux":
		return "/var/lib/spore-os/hub/spored.manifest.spore.yaml"
	case "windows":
		return `C:\ProgramData\spore-os\hub\spored.manifest.spore.yaml`
	default:
		if exePath, err := os.Executable(); err == nil {
			return filepath.Join(filepath.Dir(exePath), "spored.manifest.spore.yaml")
		}
		return "spored.manifest.spore.yaml"
	}
}
