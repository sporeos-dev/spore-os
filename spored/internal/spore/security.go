package spore

import "spored/internal/utilities/error"

func (s *Spore) securityKeyringList(request icast) *error.Error {
	return error.New(
		error.NotImplemented,
		error.Spore,
		"spore security keyring list not yet implemented")
}

func (s *Spore) securityKeyringInfo(request icast) *error.Error {
	return error.New(
		error.NotImplemented,
		error.Spore,
		"spore security keyring info not yet implemented")
}

func (s *Spore) securityKeyringGrant(request icast) *error.Error {
	return error.New(
		error.NotImplemented,
		error.Spore,
		"spore security keyring grant not yet implemented")
}

func (s *Spore) securityKeyringRevoke(request icast) *error.Error {
	return error.New(
		error.NotImplemented,
		error.Spore,
		"spore security keyring revoke not yet implemented")
}

func (s *Spore) securitySignatureSign(request icast) *error.Error {
	return error.New(
		error.NotImplemented,
		error.Spore,
		"spore security signature sign not yet implemented")
}

func (s *Spore) securitySignatureVerify(request icast) *error.Error {
	return error.New(
		error.NotImplemented,
		error.Spore,
		"spore security signature verify not yet implemented")
}
