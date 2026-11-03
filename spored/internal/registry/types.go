// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package registry

type imanifest interface {
	GetId() string
	GetName() string
	GetTrust() string
	GetManifestPath() string
	GetManifestChecksum() string
	GetBinaryPath() string
}
