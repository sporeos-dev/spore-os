package spore

import (
	"spored/internal/iface"
	"spored/internal/message"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"spored/internal/utilities/parse"
)

func (s *Spore) permissionRequest(request iface.Message) *error.Error {
	
	node, ok := request.Arg("node")
	if !ok {
		return error.MissingArg("node", error.Spore).WithMessage(request)
	}

	cap, ok := request.Arg("capability")
	if !ok {
		return error.MissingArg("capability", error.Spore).WithMessage(request)
	}

	reasonsStr := request.ArgIf("reasons", `[ "No reasons were provided for this request", "Our suggestion is to deny this request" ] `)
	reasons := parse.ArrayToStrings(reasonsStr)

	go func() {
		granted, err := s.permissions.Request(node, cap, reasons)
		if err != nil {
			s.bus.Response(err.WithMessage(request))
			return
		}

		value := "no"
		if granted {
			value = "yes"
		}

		s.bus.Response(
			message.Spore(
				request,
				out.Flag(value)))
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
