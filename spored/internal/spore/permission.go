// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package spore

import (
	"fmt"
	"spored/internal/iface"
	"spored/internal/message"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"spored/internal/utilities/parse"
)

func (s *Spore) permissionRequest(request iface.Message) *error.Error {
	cap, ok := request.Arg("capability")
	if !ok {
		return error.MissingArg("capability", error.Spore).WithMessage(request)
	}
	node := request.Origin()
	cap = s.bus.FullyQualifiedRequest(cap)

	reasonsStr := request.ArgIf("reasons", `[ "No reasons were provided for this request", "Our suggestion is to deny this request" ] `)
	reasons := parse.ArrayToStrings(reasonsStr)

	go func() {
		token, granted, err := s.permissions.RequestPermit(node, cap, reasons)
		if err != nil {
			s.bus.Response(err.WithMessage(request))
			return
		}

		if !granted {
			s.bus.Response(message.Spore(request, out.Flag("no")))
			return
		}
		response, ok := message.Response(
			fmt.Sprintf(
				"~%s:%s %s ok cast=%s",
				request.Handle(),
				request.Capability(),
				out.Pair("permission.token", token).String(),
				request.Cast()),
			s.Id())
		if !ok {
			s.bus.Response(error.New(
				error.Generic,
				error.Spore,
				"unable to create permission permit response").
				WithMessage(request))
			return
		}
		s.bus.Response(response)
	}()

	return nil
}

func (s *Spore) permissionGrant(request iface.Message) *error.Error {

	node, ok := request.Arg("node")
	if !ok {
		return error.MissingArg("node", error.Spore).WithMessage(request)
	}

	cap, ok := request.Arg("capability")
	if !ok {
		return error.MissingArg("capability", error.Spore).WithMessage(request)
	}

	go func() {
		err := s.permissions.Grant(node, cap)
		if err != nil {
			s.bus.Response(err.WithMessage(request))
			return
		}

		s.bus.Response(message.Spore(request))
	}()

	return nil
}

func (s *Spore) permissionRevoke(request iface.Message) *error.Error {

	node, ok := request.Arg("node")
	if !ok {
		return error.MissingArg("node", error.Spore).WithMessage(request)
	}

	cap, ok := request.Arg("capability")
	if !ok {
		return error.MissingArg("capability", error.Spore).WithMessage(request)
	}

	go func() {
		err := s.permissions.Revoke(node, cap)
		if err != nil {
			s.bus.Response(err.WithMessage(request))
			return
		}

		s.bus.Response(message.Spore(request))
	}()

	return nil
}
