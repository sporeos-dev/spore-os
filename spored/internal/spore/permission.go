package spore

import "spored/internal/utilities/error"

func (s *Spore) permissionList(request icast) *error.Error {
	return error.New(error.RouteNotImplemented, "not yet impl")
}

func (s *Spore) permissionRequest(request icast) *error.Error {
	return error.New(error.RouteNotImplemented, "not yet impl")
}

func (s *Spore) permissionGrant(request icast) *error.Error {
	return error.New(error.RouteNotImplemented, "not yet impl")
}

func (s *Spore) permissionRevoke(request icast) *error.Error {
	return error.New(error.RouteNotImplemented, "not yet impl")
}
