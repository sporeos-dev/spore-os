package spore

import (
	"fmt"
	"os"
	"path/filepath"
	"spored/internal/message"
	"spored/internal/registry"
	"strings"
)

// publishEvent constructs and broadcasts a lifecycle topic event.
// Best-effort — failures are logged to witness but don't affect the caller.
func (s *Spore) publishEvent(topic string, args ...string) {
	if s.broadcaster == nil {
		return
	}
	raw := "publish " + topic
	for _, arg := range args {
		raw += " " + arg
	}
	pub, err := message.Parse(raw, "dev.sporeos.SPORE")
	if err != nil {
		s.witness.Spore(message.SporeEvent("warn", "Failed to build publish event", "topic="+topic))
		return
	}
	if t, ok := pub.(message.Topic); ok {
		if err := s.broadcaster.Publish(t); err != nil {
			s.witness.Spore(message.SporeEvent("warn", "Failed to publish event", "topic="+topic))
		}
	}
}

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