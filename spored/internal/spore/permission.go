package spore

import "spored/internal/utilities/error"

func (s *Spore) permissionList(request icast) *error.Error {
	return error.New(
		error.NotImplemented,
		error.Spore,
		"spore permission list not yet implemented")
}

func (s *Spore) permissionRequest(request icast) *error.Error {
	return error.New(
		error.NotImplemented,
		error.Spore,
		"spore permission request not yet implemented")
}

func (s *Spore) permissionGrant(request icast) *error.Error {
	return error.New(
		error.NotImplemented,
		error.Spore,
		"spore permission grant not yet implemented")
}

func (s *Spore) permissionRevoke(request icast) *error.Error {
	return error.New(
		error.NotImplemented,
		error.Spore,
		"spore permission revoke not yet implemented")
}
