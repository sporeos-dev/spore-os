package spore

import (
	"fmt"
	"os"
	"path/filepath"
	"spored/internal/message"
	"spored/internal/registry"
	"strings"
)

func (s *Spore) returnCapture(msg *message.Spore, args map[string]string, flags []string) string {
	parts := []string{ fmt.Sprintf("~%s:%s ok capture=dev.sporeos.SPORE", msg.Handle(), msg.Command()) }
	parts = append(parts, flags...)
	for k, v := range args {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, " ")
}

// hubManifestPath returns the system-level path for the hub's own manifest.
// The manifest lives at the data root alongside nodes.registry.yaml.
func hubManifestPath() string {
	if root, err := registry.DataRoot(); err == nil {
		return filepath.Join(root, "spored.manifest.spore.yaml")
	}
	// Fallback for unsupported platforms — look next to the binary.
	if exePath, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(exePath), "spored.manifest.spore.yaml")
	}
	return "spored.manifest.spore.yaml"
}