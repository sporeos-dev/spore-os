// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build windows

package nodes

import (
	"testing"

	"spored/internal/pal"
	"spored/internal/utilities/status"
)

func TestBinaryVerificationDefersPeerPathAccessDeniedToUserSpace(t *testing.T) {
	binary := &binary{status: status.New()}
	binary.setPeerPathFailureStatus(pal.ErrPeerProcessPathAccessDenied)

	if binary.status.Get() != status.RequiresUserSpace {
		t.Fatalf("binary status = %s, want %s", binary.status.String(), status.RequiresUserSpace)
	}
}
