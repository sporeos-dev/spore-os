// Copyright 2026 mharr
// SPDX-License-Identifier: AGPL-3.0-only

package hub

import (
	"bufio"
	"net"
	"strings"
	"testing"
)

// =============================================================================
// SPEC §4: Handshake
// "When a node starts, it opens a connection to the hub socket and sends:
//     com.example.clock
//  The hub responds:
//     OK"
// =============================================================================

func TestHandshake_ReturnsNodeIDAndOK(t *testing.T) {
	// Create a socket pair for testing
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	done := make(chan struct{})
	var result string
	var handshakeErr error

	go func() {
		defer close(done)
		result, handshakeErr = shakeHands(server, func(string) bool { return true })
	}()
	_, err := client.Write([]byte("com.example.clock\n"))
	if err != nil {
		t.Fatalf("client write failed: %v", err)
	}

	// Read hub response
	reader := bufio.NewReader(client)
	response, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("failed to read hub response: %v", err)
	}

	<-done

	if handshakeErr != nil {
		t.Fatalf("handshake error: %v", handshakeErr)
	}

	// SPEC §4: Hub responds with "OK"
	if strings.TrimSpace(response) != "OK" {
		t.Errorf("expected OK response, got %q", strings.TrimSpace(response))
	}

	// The returned result should be the node ID
	if result != "com.example.clock" {
		t.Errorf("expected node ID com.example.clock, got %q", result)
	}
}

func TestHandshake_TrimsWhitespace(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	done := make(chan struct{})
	var result string
	var handshakeErr error

	go func() {
		defer close(done)
		result, handshakeErr = shakeHands(server, func(string) bool { return true })
	}()

	// Send ID with extra whitespace
	_, _ = client.Write([]byte("  com.example.clock  \n"))

	reader := bufio.NewReader(client)
	_, _ = reader.ReadString('\n')
	<-done

	if handshakeErr != nil {
		t.Fatalf("handshake error: %v", handshakeErr)
	}
	if result != "com.example.clock" {
		t.Errorf("expected trimmed ID, got %q", result)
	}
}

func TestHandshake_FailsOnClosedConnection(t *testing.T) {
	server, client := net.Pipe()
	client.Close() // Close immediately

	_, err := shakeHands(server, func(string) bool { return true })
	server.Close()

	if err == nil {
		t.Error("expected error when connection is closed before handshake")
	}
}

func TestHandshake_ReverseDomainNodeID(t *testing.T) {
	// SPEC §8.2: id is a "Reverse-domain unique identifier"
	testIDs := []string{
		"com.example.clock",
		"com.google.filesystem",
		"dev.sporeos.myapp",
		"org.mozilla.browser",
	}

	for _, id := range testIDs {
		server, client := net.Pipe()

		done := make(chan struct{})
		var result string
		var handshakeErr error

		go func() {
			defer close(done)
			result, handshakeErr = shakeHands(server, func(string) bool { return true })
		}()

		_, _ = client.Write([]byte(id + "\n"))
		reader := bufio.NewReader(client)
		_, _ = reader.ReadString('\n')
		<-done

		server.Close()
		client.Close()

		if handshakeErr != nil {
			t.Errorf("handshake failed for ID %q: %v", id, handshakeErr)
			continue
		}
		if result != id {
			t.Errorf("expected %q, got %q", id, result)
		}
	}
}

func TestHandshake_RejectsUnknownNodeID(t *testing.T) {
	// SPEC §4: "If the node ID is not known, the hub responds with an error."
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	done := make(chan struct{})
	var handshakeErr error

	go func() {
		defer close(done)
		_, handshakeErr = shakeHands(server, func(string) bool { return false })
	}()

	_, _ = client.Write([]byte("com.unknown.node\n"))

	reader := bufio.NewReader(client)
	response, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("failed to read hub response: %v", err)
	}

	<-done

	// Hub must respond with an error (not OK) and handshake must fail.
	if strings.TrimSpace(response) == "OK" {
		t.Error("expected error response for unknown node, got OK")
	}
	if !strings.Contains(response, "error") {
		t.Errorf("expected error in response, got %q", strings.TrimSpace(response))
	}
	if handshakeErr == nil {
		t.Error("expected handshake to return an error for unknown node ID")
	}
}
