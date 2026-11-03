// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package message

import "strings"

// CastIdentity distinguishes the hub component routing a request from the
// original node that initiated it.
type CastIdentity struct {
	Actor  string
	Origin string
}

func castIdentity(cast string) CastIdentity {
	actor, origin, ok := strings.Cut(cast, ":")
	if !ok {
		return CastIdentity{Actor: cast, Origin: cast}
	}
	return CastIdentity{Actor: actor, Origin: origin}
}

func DelegatedCast(actor string, origin string) string {
	if actor == origin {
		return actor
	}
	return actor + ":" + origin
}
