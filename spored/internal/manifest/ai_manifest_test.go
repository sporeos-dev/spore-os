// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package manifest

import (
	"spored/internal/utilities/status"
	"testing"
)

func TestVerifyDeveloperManifestDoesNotRequireLocalReadAccess(t *testing.T) {
	m := &Manifest{
		Trust:  Developer,
		Path:   "/unavailable/developer.manifest.spore.yaml",
		Status: status.New(),
	}

	m.Verify()

	if got := m.Status.Get(); got != status.RequiresDeveloper {
		t.Fatalf("developer manifest status = %s, want %s", got, status.RequiresDeveloper)
	}
}

func TestVerifyDoesNotReplaceMissingStatus(t *testing.T) {
	m := &Manifest{
		Trust:  Developer,
		Path:   "/missing/developer.manifest.spore.yaml",
		Status: status.New(),
	}
	m.Status.Set(status.Missing)

	m.Verify()

	if got := m.Status.Get(); got != status.Missing {
		t.Fatalf("missing manifest status = %s, want %s", got, status.Missing)
	}
}
