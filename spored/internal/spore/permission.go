package spore

import (
	"spored/internal/message"
	"spored/internal/utilities/error"
)

func (s *Spore) permissionList(request message.Message) *error.Error {
	return error.New(
		error.NotImplemented,
		error.Spore,
		"spore permission list not yet implemented")
}

func (s *Spore) permissionRequest(request message.Message) *error.Error {
	return error.New(
		error.NotImplemented,
		error.Spore,
		"spore permission request not yet implemented")
}

func (s *Spore) permissionGrant(request message.Message) *error.Error {
	return error.New(
		error.NotImplemented,
		error.Spore,
		"spore permission grant not yet implemented")
}

func (s *Spore) permissionRevoke(request message.Message) *error.Error {
	return error.New(
		error.NotImplemented,
		error.Spore,
		"spore permission revoke not yet implemented")
}
