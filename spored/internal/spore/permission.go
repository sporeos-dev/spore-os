package spore

import (
	"spored/internal/message"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"spored/internal/utilities/parse"
)

func (s *Spore) permissionList(request message.Message) *error.Error {

	nodeid, ok := request.Arg("node")
	if !ok {
		return error.MissingArg("node", error.Spore).WithMessage(request)
	}

	go func() {
		permissions, err := s.permissions.List(nodeid)
		if err != nil {
			s.bus.Response(err.WithMessage(request))
			return
		}

		s.bus.Response(
			message.Spore(
				request,
				out.Array("permissions", permissions)))
	}()

	return nil
}

func (s *Spore) permissionRequest(request message.Message) *error.Error {
	
	node, ok := request.Arg("node")
	if !ok {
		return error.MissingArg("node", error.Spore).WithMessage(request)
	}

	cap, ok := request.Arg("capability")
	if !ok {
		return error.MissingArg("capability", error.Spore).WithMessage(request)
	}

	reasonsStr := request.ArgIf("reasons", `["Reasons: not provided", "Suggestion: deny"]`)
	reasons := parse.ArrayToStrings(reasonsStr)

	go func() {
		value, err := s.permissions.Request(node, cap, reasons)
		if err != nil {
			s.bus.Response(err.WithMessage(request))
			return
		}

		s.bus.Response(
			message.Spore(
				request,
				out.Flag(string(value))))
	}()
	
	return nil
}

func (s *Spore) permissionGrant(request message.Message) *error.Error {
	
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

func (s *Spore) permissionRevoke(request message.Message) *error.Error {

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
