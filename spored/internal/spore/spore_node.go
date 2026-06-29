package spore

import (
	"encoding/json"
	"errors"
	"spored/internal/manifest"
	"spored/internal/message"
)

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