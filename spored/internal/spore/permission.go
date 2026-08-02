package spore

import (
	"spored/internal/message"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"spored/internal/utilities/parse"
)

func (s *Spore) permissionList(request message.Message) *error.Error {

	nodeid, err := request.Arg("node")
	if err != nil {
		return err
	}

	go func() {
		permissions, err := s.permissions.List(nodeid)
		if err != nil {
			s.respondError(request, err)
		}

		s.respond(
			request,
			out.Array("permissions", permissions))
	}()

	return nil
}

func (s *Spore) permissionRequest(request message.Message) *error.Error {
	
	node, err := request.Arg("node")
	if err != nil {
		return err
	}

	cap, err := request.Arg("capability")
	if err != nil {
		return err
	}

	reasonsStr := request.ArgIf("reasons", `["Reasons: not provided", "Suggestion: deny"]`)
	reasons := parse.ArrayToStrings(reasonsStr)

	go func() {
		value, err := s.permissions.Request(node, cap, reasons)
		if err != nil {
			s.respondError(request, err)
			return
		}

		s.respond(
			request,
			out.Flag(string(value)))
	}()
	
	return nil
}

func (s *Spore) permissionGrant(request message.Message) *error.Error {
	
	node, err := request.Arg("node")
	if err != nil {
		return err
	}

	cap, err := request.Arg("capability")
	if err != nil {
		return err
	}

	go func() {
		
		err := s.permissions.Grant(node, cap)
		if err != nil {
			s.respondError(request, err)
			return
		}

		s.respond(request)
	}()

	return nil
}

func (s *Spore) permissionRevoke(request message.Message) *error.Error {

	node, err := request.Arg("node")
	if err != nil {
		return err
	}

	cap, err := request.Arg("capability")
	if err != nil {
		return err
	}

	go func() {
		
		err := s.permissions.Revoke(node, cap)
		if err != nil {
			s.respondError(request, err)
			return
		}

		s.respond(request)
	}()

	return nil
}
