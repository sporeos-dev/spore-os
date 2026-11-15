// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package manifestloader

import (
	"spored/internal/hyphae"
	"spored/internal/manifest"
	"spored/internal/utilities/error"
	"spored/internal/utilities/file"
	"spored/internal/utilities/out"
	"spored/internal/witness"
)

func Load(m *manifest.Manifest) bool {
	var contents string
	if file.IsReadable(m.Path) {
		content, err := file.Read(m.Path)
		if err != nil {
			witness.Error(error.RuntimeFailure, error.Manifest, "read failed", out.Pair("path", m.Path), out.Pair("error", err.Error()))
			return false
		}
		if content == "" {
			witness.Error(error.RuntimeFailure, error.Manifest, "empty manifest", out.Pair("path", m.Path))
			return false
		}
		contents = content
	} else {
		var ok bool
		contents, ok = hyphae.ManifestRead(m.Path)
		if !ok {
			witness.Error(error.RuntimeFailure, error.Manifest, "user-space read failed", out.Pair("path", m.Path))
			return false
		}
	}

	if err := m.LoadContent(contents); err != nil {
		witness.Error(error.Malformed, error.Manifest, "invalid manifest", out.Pair("path", m.Path), out.Pair("error", err.Error()))
		return false
	}
	return true
}
