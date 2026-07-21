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

func ElementFromManifest(manifest imanifest) *Element {
	el := &Element {
		ID: manifest.GetId(),
		Name: manifest.GetName(),
		Manifest: manifest.GetManifestPath(),
		Checksum: manifest.GetManifestChecksum(),
		Binary: manifest.GetBinaryPath(),
	}

	checksum, err := file.CalculateChecksum(el.Binary)
	if err != nil {
		return nil
	}
	el.BinaryChecksum = checksum
	
	return el
}