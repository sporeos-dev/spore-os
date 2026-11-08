// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package spore

import (
	"spored/internal/bus"
	"spored/internal/cparser"
	"spored/internal/manifest"
	"testing"
)

func TestNodeSpawnMissingNodeReturnsError(t *testing.T) {
	request := mustNodeCommandRequest(t, "dev.sporeos.SPORE.node.spawn ~missing-node")
	router := bus.New()
	router.Set(nil, nil, &permissionRequestTestPermissions{}, nil)
	s := &Spore{
		manifest: &manifest.Manifest{
			ID:  "dev.sporeos.SPORE",
			Api: []manifest.Command{{Name: "node.spawn"}},
		},
		bus: router,
	}
	router.Register(s)
	err := router.Request(request)
	if err == nil {
		t.Fatal("expected missing node argument error")
	}
	reply, ok := cparser.Validate(err.Wire())
	if !ok {
		t.Fatalf("invalid error response: %q", err.Wire())
	}
	if reply.Handle != "missing-node" || reply.Args["code"] != "Missing.Spore" || reply.Args["argument"] != "node" {
		t.Errorf("unexpected missing argument response: %q", err.Wire())
	}
}
