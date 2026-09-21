// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package spore

import (
	"spored/internal/iface"
	"spored/internal/message"
	"spored/internal/utilities/error"
)

func (s *Spore) nodeInstall(request iface.Message) *error.Error {
	
	path, ok := request.Arg("path")
	if !ok {
		return error.MissingArg("path", error.Spore).WithMessage(request)
	}

	go func() {
		err := s.nodes.Install(path)
		if err != nil {
			s.bus.Response(err.WithMessage(request))
		} else {
			s.bus.Response(message.Spore(request))
		}
	}()

	return nil
}

func (s *Spore) nodeUninstall(request iface.Message) *error.Error {
	
	node, ok := request.Arg("node")
	if !ok {
		return error.MissingArg("node", error.Spore).WithMessage(request)
	}

	go func() {
		err := s.nodes.Uninstall(node)
		if err != nil {
			s.bus.Response(err.WithMessage(request))
		} else {
			s.bus.Response(message.Spore(request))
		}
	}()

	return nil
}

func (s *Spore) nodeSpawn(request iface.Message) *error.Error {

	node, ok := request.Arg("node")
	if !ok {
		return error.MissingArg("node", error.Spore).WithMessage(request)
	}

	go func() {
		err := s.nodes.Spawn(node)
		if err != nil {
			s.bus.Response(err.WithMessage(request))
		} else {
			s.bus.Response(message.Spore(request))
		}
	}()

	return nil
}

func (s *Spore) nodeKill(request iface.Message) *error.Error {

	node, ok := request.Arg("node")
	if !ok {
		return error.MissingArg("node", error.Spore).WithMessage(request)
	}

	go func() {
		err := s.nodes.Kill(node)
		if err != nil {
			s.bus.Response(err.WithMessage(request))
		} else {
			s.bus.Response(message.Spore(request))
		}
	}()

	return nil
}
