package spore

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"spored/internal/manifest"
	"spored/internal/message"
	"spored/internal/utilities"
)

const hyPhaeNodeID = "dev.sporeos.hyphae"

func (s *Spore) nodeInstall(msg *message.Spore) (string, error) {
	path, err := msg.GetArgument("path")
	if err != nil {
		return "", err
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("cannot resolve path: %w", err)
	}
	path = absPath

	hypha, hyErr := s.hub.GetNode(hyPhaeNodeID)
	if hyErr != nil || !hypha.IsConnected() {
		return "", fmt.Errorf("hyphae is required but not available")
	}

	s.witness.Spore(message.SporeEvent("info", "Delegating install to hyphae", "path="+path))
	parsedManifest, err := s.nodeInstallViaHyphae(path)
	if err != nil {
		return "", err
	}

	// Load the node into memory immediately so it's usable without a restart.
	if err = s.hub.AddNodeWithManifest(parsedManifest); err != nil {
		s.registry.Remove(path)
		return "", err
	}

	// TODO: add a flag or config to allow direct (non-hyphae) installs
	// var parsedManifest *manifest.Manifest
	// hypha, hyErr := s.hub.GetNode(hyPhaeNodeID)
	// if hyErr == nil && hypha.IsConnected() {
	// 	slog.Info("nodeInstall: delegating to hyphae", "path", path)
	// 	var m *manifest.Manifest
	// 	m, err = s.nodeInstallViaHyphae(path)
	// 	if err == nil {
	// 		parsedManifest = m
	// 	}
	// } else {
	// 	if hyErr != nil {
	// 		slog.Warn("nodeInstall: hyphae not registered, falling back to direct reads", "error", hyErr)
	// 	} else {
	// 		slog.Warn("nodeInstall: hyphae registered but not connected, falling back to direct reads")
	// 	}
	// 	err = s.registry.Add(path)
	// }
	// if err != nil {
	// 	return "", err
	// }
	// if parsedManifest != nil {
	// 	if err = s.hub.AddNodeWithManifest(parsedManifest); err != nil {
	// 		s.registry.Remove(path)
	// 		return "", err
	// 	}
	// } else {
	// 	if err = s.hub.AddNode(path); err != nil {
	// 		s.registry.Remove(path)
	// 		return "", err
	// 	}
	// 	parsedManifest, _ = manifest.LoadManifest(path)
	// }
	s.witness.Spore(message.SporeEvent("info", "Node installed", "path="+path))
	if parsedManifest != nil {
		s.publishEvent("SPORE.node.installed", "node="+parsedManifest.ID)
	}
	return s.returnCapture(msg, map[string]string{}, []string{}), nil
}

// nodeInstallViaHyphae registers a node by delegating file reads and hashing
// to spore-hyphae. This is used when the hub daemon lacks direct read
// permission to files in the user's session (e.g. files under ~/Library or
// a user's home directory). It returns the parsed manifest so the caller can
// load the node into memory without a second direct file read.
func (s *Spore) nodeInstallViaHyphae(manifestPath string) (*manifest.Manifest, error) {
	// 1. Read the manifest content.
	readReply, err := s.router.RequestNode(hyPhaeNodeID, "HYPHAE.manifest.read",
		map[string]string{"path": manifestPath})
	if err != nil {
		return nil, fmt.Errorf("install via hyphae: read manifest: %w", err)
	}
	cap, ok := readReply.(*message.Capture)
	if !ok {
		return nil, fmt.Errorf("install via hyphae: unexpected reply type from HYPHAE.manifest.read")
	}
	content, ok := cap.GetArg("content")
	if !ok {
		return nil, fmt.Errorf("install via hyphae: HYPHAE.manifest.read returned no content field")
	}

	// 2. Hash the manifest file.
	manifestHashReply, err := s.router.RequestNode(hyPhaeNodeID, "HYPHAE.file.hash",
		map[string]string{"path": manifestPath})
	if err != nil {
		return nil, fmt.Errorf("install via hyphae: hash manifest: %w", err)
	}
	manifestHashCap, ok := manifestHashReply.(*message.Capture)
	if !ok {
		return nil, fmt.Errorf("install via hyphae: unexpected reply type from HYPHAE.file.hash (manifest)")
	}
	manifestHash, ok := manifestHashCap.GetArg("hash")
	if !ok {
		return nil, fmt.Errorf("install via hyphae: HYPHAE.file.hash returned no hash field")
	}
	manifestChecksum := "sha256:" + manifestHash

	// 3. Parse the manifest to find the binary path.
	m, err := manifest.ParseManifest(content)
	if err != nil {
		return nil, fmt.Errorf("install via hyphae: parse manifest: %w", err)
	}
	m.Path = manifestPath

	// 4. Hash the binary if one is declared.
	binaryPath := ""
	binaryChecksum := ""
	if m.App != "n/a" && m.App != "" {
		binaryPath = m.App
		if !filepath.IsAbs(binaryPath) {
			binaryPath = filepath.Join(filepath.Dir(manifestPath), binaryPath)
		}
		binaryHashReply, err := s.router.RequestNode(hyPhaeNodeID, "HYPHAE.file.hash",
			map[string]string{"path": binaryPath})
		if err != nil {
			return nil, fmt.Errorf("install via hyphae: hash binary: %w", err)
		}
		binaryHashCap, ok := binaryHashReply.(*message.Capture)
		if !ok {
			return nil, fmt.Errorf("install via hyphae: unexpected reply type from HYPHAE.file.hash (binary)")
		}
		binaryHash, ok := binaryHashCap.GetArg("hash")
		if !ok {
			return nil, fmt.Errorf("install via hyphae: HYPHAE.file.hash returned no hash field")
		}
		binaryChecksum = "sha256:" + binaryHash
	}

	if err := s.registry.AddEntry(manifestPath, content, manifestChecksum, binaryPath, binaryChecksum); err != nil {
		return nil, err
	}
	return m, nil
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
	if nodeid != "n/a" {
		s.publishEvent("SPORE.node.uninstalled", "node="+nodeid)
	}
	return s.returnCapture(msg, map[string]string{}, []string{}), nil
}

func (s *Spore) nodeSpawn(msg *message.Spore) (string, error) {
	nodeid, err := msg.GetArgument("node")
	if err != nil {
		return "", err
	}

	hypha, hyErr := s.hub.GetNode(hyPhaeNodeID)
	if hyErr != nil || !hypha.IsConnected() {
		return "", fmt.Errorf("hyphae is required but not available")
	}

	s.witness.Spore(message.SporeEvent("info", "Delegating spawn to hyphae", "node="+nodeid))
	if err := s.nodeSpawnViaHyphae(nodeid); err != nil {
		return "", err
	}

	// TODO: add a flag or config to allow direct (non-hyphae) spawns
	// hypha, hyErr := s.hub.GetNode(hyPhaeNodeID)
	// if hyErr == nil && hypha.IsConnected() {
	// 	slog.Info("nodeSpawn: delegating to hyphae", "node", nodeid)
	// 	if err := s.nodeSpawnViaHyphae(nodeid); err != nil {
	// 		return "", err
	// 	}
	// } else {
	// 	if hyErr != nil {
	// 		slog.Warn("nodeSpawn: hyphae not registered, falling back to direct spawn", "error", hyErr)
	// 	} else {
	// 		slog.Warn("nodeSpawn: hyphae registered but not connected, falling back to direct spawn")
	// 	}
	// 	if err := s.hub.SpawnNode(nodeid); err != nil {
	// 		return "", err
	// 	}
	// }

	s.witness.Spore(message.SporeEvent("info", "Node spawned", "node="+nodeid))
	s.publishEvent("SPORE.node.spawned", "node="+nodeid)
	return s.returnCapture(msg, map[string]string{}, []string{}), nil
}

// nodeSpawnViaHyphae resolves the node's binary path and delegates execution
// to spore-hyphae, which runs in the user session. This is preferred over a
// direct exec when the hub daemon runs outside the user session (e.g. as a
// launch daemon on macOS) because the spawned node inherits the user's
// environment rather than the daemon's environment.
func (s *Spore) nodeSpawnViaHyphae(nodeid string) error {
	node, err := s.hub.GetNode(nodeid)
	if err != nil {
		return err
	}

	if node.IsConnected() {
		return errors.New("node is already running")
	}

	m := node.GetManifest()
	if m.App == "" || m.App == "n/a" {
		return errors.New("node has no executable defined")
	}

	binaryPath, err := utilities.ResolveAppPath(m.App, filepath.Dir(m.Path))
	if err != nil {
		return fmt.Errorf("spawn via hyphae: resolve binary path: %w", err)
	}

	_, err = s.router.RequestNode(hyPhaeNodeID, "HYPHAE.node.spawn",
		map[string]string{"binary": binaryPath})
	if err != nil {
		return fmt.Errorf("spawn via hyphae: %w", err)
	}
	return nil
}

func (s *Spore) nodeKill(msg *message.Spore) (string, error) {
	nodeid, err := msg.GetArgument("node")
	if err != nil {
		return "", err
	}

	hypha, hyErr := s.hub.GetNode(hyPhaeNodeID)
	if hyErr != nil || !hypha.IsConnected() {
		return "", fmt.Errorf("hyphae is required but not available")
	}

	s.witness.Spore(message.SporeEvent("info", "Delegating kill to hyphae", "node="+nodeid))
	if err := s.nodeKillViaHyphae(nodeid); err != nil {
		return "", err
	}

	// TODO: add a flag or config to allow direct (non-hyphae) kills
	// hypha, hyErr := s.hub.GetNode(hyPhaeNodeID)
	// if hyErr == nil && hypha.IsConnected() {
	// 	slog.Info("nodeKill: delegating to hyphae", "node", nodeid)
	// 	if err := s.nodeKillViaHyphae(nodeid); err != nil {
	// 		return "", err
	// 	}
	// } else {
	// 	if hyErr != nil {
	// 		slog.Warn("nodeKill: hyphae not registered, falling back to direct kill", "error", hyErr)
	// 	} else {
	// 		slog.Warn("nodeKill: hyphae registered but not connected, falling back to direct kill")
	// 	}
	// 	if err := s.hub.KillNode(nodeid); err != nil {
	// 		return "", err
	// 	}
	// }

	s.witness.Spore(message.SporeEvent("info", "Node killed", "node="+nodeid))
	s.publishEvent("SPORE.node.killed", "node="+nodeid)
	return s.returnCapture(msg, map[string]string{}, []string{}), nil
}

// nodeKillViaHyphae delegates process termination to spore-hyphae, which runs
// in the user session and can signal processes that were spawned there.
func (s *Spore) nodeKillViaHyphae(nodeid string) error {
	node, err := s.hub.GetNode(nodeid)
	if err != nil {
		return err
	}

	if !node.IsConnected() {
		return errors.New("node is not running")
	}

	pid := node.GetPID()
	if pid == 0 {
		return errors.New("peer PID not available on this platform")
	}

	_, err = s.router.RequestNode(hyPhaeNodeID, "HYPHAE.node.kill",
		map[string]string{"pid": fmt.Sprintf("%d", pid)})
	if err != nil {
		return fmt.Errorf("kill via hyphae: %w", err)
	}
	return nil
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