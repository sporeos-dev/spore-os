package spore

import (
	"os"
	"path/filepath"
	"spored/internal/manifest"
	"spored/internal/message"
)

func (s *Spore) developerManifestReplace(msg *message.Spore) (string, error) {
	path, err := msg.GetArgument("path")
	if err != nil {
		return "", err
	}

	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		// System path not found — fall back to next to binary (dev / direct run).
		if exePath, err := os.Executable(); err == nil {
			path = filepath.Join(filepath.Dir(exePath), "spored.manifest.spore.yaml")
		}
	}

	m, err := manifest.LoadManifest(path)
	if err != nil {
		return "", err
	}
	s.manifest = m

	return s.returnCapture(msg, map[string]string{}, []string{}), nil
}
