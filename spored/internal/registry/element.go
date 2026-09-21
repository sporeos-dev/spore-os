// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package registry

import "spored/internal/utilities/file"

type Element struct {
	ID			   string `yaml:"id"`
	Name           string `yaml:"name"`
	Manifest       string `yaml:"manifest"`
	Checksum       string `yaml:"checksum"`
	Binary         string `yaml:"binary"`
	BinaryChecksum string `yaml:"binaryChecksum"`
}

func ElementFromManifest(man imanifest) *Element {
	el := &Element {
		ID: man.GetId(),
		Name: man.GetName(),
		Manifest: man.GetManifestPath(),
		Checksum: man.GetManifestChecksum(),
		Binary: man.GetBinaryPath(),
	}

	if man.GetTrust() == "developer" {
		el.Checksum = "developer"
		el.BinaryChecksum = "developer"
	} else {
		checksum, err := file.CalculateChecksum(el.Binary)
		if err != nil {
			return nil
		}
		el.BinaryChecksum = checksum
	}
	
	return el
}