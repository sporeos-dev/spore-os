// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package spore

import (
	"testing"
	"time"

	"spored/internal/bus"
	"spored/internal/iface"
	"spored/internal/manifest"
	"spored/internal/message"
	"spored/internal/nodes"
	sporeerror "spored/internal/utilities/error"
)

type nodeCommandTestBus struct {
	responses chan iface.Message
}

func (b *nodeCommandTestBus) Register(bus.INode)   {}
func (b *nodeCommandTestBus) Unregister(bus.INode) {}
func (b *nodeCommandTestBus) Subscribe(string, string) *sporeerror.Error {
	return nil
}
func (b *nodeCommandTestBus) Unsubscribe(string, string) *sporeerror.Error {
	return nil
}
func (b *nodeCommandTestBus) Response(response iface.Message) *sporeerror.Error {
	b.responses <- response
	return nil
}
func (b *nodeCommandTestBus) FullyQualifiedRequest(command string) string {
	return command
}
func (b *nodeCommandTestBus) Witness(iface.Message) {}

type nodeCommandTestNodes struct {
	installPath   string
	installLoad   bool
	reinstallNode string
	reinstallLoad bool
	reloadNode    string
}

func (n *nodeCommandTestNodes) GetNodes() []string { return nil }

func (n *nodeCommandTestNodes) GetManifest(string) *manifest.Manifest { return nil }

func (n *nodeCommandTestNodes) Install(path string, load bool) *sporeerror.Error {
	n.installPath = path
	n.installLoad = load
	return nil
}

func (n *nodeCommandTestNodes) Uninstall(string) *sporeerror.Error { return nil }

func (n *nodeCommandTestNodes) Reinstall(node string, load bool) *sporeerror.Error {
	n.reinstallNode = node
	n.reinstallLoad = load
	return nil
}

func (n *nodeCommandTestNodes) Spawn(string) *sporeerror.Error { return nil }

func (n *nodeCommandTestNodes) Kill(string) *sporeerror.Error { return nil }

func (n *nodeCommandTestNodes) Reload(node string) *sporeerror.Error {
	n.reloadNode = node
	return nil
}

func (n *nodeCommandTestNodes) GetState(string) (*nodes.State, *sporeerror.Error) {
	return nil, nil
}

func TestNodeInstallPassesLoadFlag(t *testing.T) {
	testNodes := &nodeCommandTestNodes{}
	testBus := &nodeCommandTestBus{responses: make(chan iface.Message, 1)}
	s := &Spore{nodes: testNodes, bus: testBus}

	request := mustNodeCommandRequest(t, "dev.sporeos.SPORE.node.install path=/tmp/example.yaml load ~install")
	if err := s.nodeInstall(request); err != nil {
		t.Fatalf("nodeInstall returned an error: %v", err)
	}
	awaitNodeCommandResponse(t, testBus)

	if testNodes.installPath != "/tmp/example.yaml" {
		t.Errorf("install path = %q, want %q", testNodes.installPath, "/tmp/example.yaml")
	}
	if !testNodes.installLoad {
		t.Fatal("install did not receive the load flag")
	}
}

func TestNodeReinstallPassesNodeAndLoadFlag(t *testing.T) {
	testNodes := &nodeCommandTestNodes{}
	testBus := &nodeCommandTestBus{responses: make(chan iface.Message, 1)}
	s := &Spore{nodes: testNodes, bus: testBus}

	request := mustNodeCommandRequest(t, "dev.sporeos.SPORE.node.reinstall node=dev.sporeos.example load ~reinstall")
	if err := s.nodeReinstall(request); err != nil {
		t.Fatalf("nodeReinstall returned an error: %v", err)
	}
	awaitNodeCommandResponse(t, testBus)

	if testNodes.reinstallNode != "dev.sporeos.example" {
		t.Errorf("reinstall node = %q, want %q", testNodes.reinstallNode, "dev.sporeos.example")
	}
	if !testNodes.reinstallLoad {
		t.Fatal("reinstall did not receive the load flag")
	}
}

func TestNodeReloadPassesNode(t *testing.T) {
	testNodes := &nodeCommandTestNodes{}
	testBus := &nodeCommandTestBus{responses: make(chan iface.Message, 1)}
	s := &Spore{nodes: testNodes, bus: testBus}

	request := mustNodeCommandRequest(t, "dev.sporeos.SPORE.node.reload node=dev.sporeos.example ~reload")
	if err := s.nodeReload(request); err != nil {
		t.Fatalf("nodeReload returned an error: %v", err)
	}
	awaitNodeCommandResponse(t, testBus)

	if testNodes.reloadNode != "dev.sporeos.example" {
		t.Errorf("reload node = %q, want %q", testNodes.reloadNode, "dev.sporeos.example")
	}
}

func mustNodeCommandRequest(t *testing.T, raw string) iface.Message {
	t.Helper()

	request, ok := message.Request(raw, "dev.sporeos.test")
	if !ok {
		t.Fatalf("failed to parse request: %q", raw)
	}
	return request
}

func awaitNodeCommandResponse(t *testing.T, bus *nodeCommandTestBus) iface.Message {
	t.Helper()

	select {
	case response := <-bus.responses:
		return response
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for node command response")
	}
	return nil
}
