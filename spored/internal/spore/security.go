// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package spore

import (
	"spored/internal/iface"
	"spored/internal/utilities/error"
)

func (s *Spore) securityKeyringGrant(request iface.Message) *error.Error {
	return error.New(
		error.NotImplemented,
		error.Spore,
		"spore security keyring grant not yet implemented").
		WithMessage(request)
}

func (s *Spore) securityKeyringRevoke(request iface.Message) *error.Error {
	return error.New(
		error.NotImplemented,
		error.Spore,
		"spore security keyring revoke not yet implemented").
		WithMessage(request)
}

func (s *Spore) securitySignatureSign(request iface.Message) *error.Error {
	return error.New(
		error.NotImplemented,
		error.Spore,
		"spore security signature sign not yet implemented").
		WithMessage(request)
}

func (s *Spore) securitySignatureVerify(request iface.Message) *error.Error {
	return error.New(
		error.NotImplemented,
		error.Spore,
		"spore security signature verify not yet implemented").
		WithMessage(request)
}
