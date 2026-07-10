// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package spore

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"spored/internal/interfaces"
	"spored/internal/manifest"
	"spored/internal/message"
)

type Spore struct {
	hub interfaces.Hub
	registry interfaces.Registry
	router interfaces.Router
	witness interfaces.Witness
	broadcaster interfaces.Broadcaster

	manifest *manifest.Manifest
}

func (s *Spore) Open(hub interfaces.Hub, registry interfaces.Registry, router interfaces.Router, witness interfaces.Witness, broadcaster interfaces.Broadcaster) {
	s.hub = hub
	s.registry = registry
	s.router = router
	s.witness = witness
	s.broadcaster = broadcaster

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
		case "SPORE.topic.list":		res, err = s.topicList(sporeMessage)
		case "SPORE.topic.help":		res, err = s.topicHelp(sporeMessage)
		case "SPORE.topic.subscribe":	res, err = s.topicSubscribe(sporeMessage)
		case "SPORE.topic.unsubscribe":	res, err = s.topicUnsubscribe(sporeMessage)
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
