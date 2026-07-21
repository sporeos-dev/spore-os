package spore

import (
	"spored/internal/message"
	"spored/internal/utilities/error"
)

func (s *Spore) securityKeyringList(request message.Message) *error.Error {
	return error.New(
		error.NotImplemented,
		error.Spore,
		"spore security keyring list not yet implemented")
}

func (s *Spore) securityKeyringInfo(request message.Message) *error.Error {
	return error.New(
		error.NotImplemented,
		error.Spore,
		"spore security keyring info not yet implemented")
}

func (s *Spore) securityKeyringGrant(request message.Message) *error.Error {
	return error.New(
		error.NotImplemented,
		error.Spore,
		"spore security keyring grant not yet implemented")
}

func (s *Spore) securityKeyringRevoke(request message.Message) *error.Error {
	return error.New(
		error.NotImplemented,
		error.Spore,
		"spore security keyring revoke not yet implemented")
}

func (s *Spore) securitySignatureSign(request message.Message) *error.Error {
	return error.New(
		error.NotImplemented,
		error.Spore,
		"spore security signature sign not yet implemented")
}

func (s *Spore) securitySignatureVerify(request message.Message) *error.Error {
	return error.New(
		error.NotImplemented,
		error.Spore,
		"spore security signature verify not yet implemented")
}
