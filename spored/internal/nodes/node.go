package nodes

import (
	"bufio"
	"errors"
	"io"
	"log/slog"
	"net"
	"spored/internal/manifest"
	"spored/internal/message"
	"spored/internal/registry"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"spored/internal/utilities/status"
	"strings"
	"sync"
)

type node struct {
	registry *registry.Element
	manifest *manifest.Manifest
	bus ibus

	mu sync.RWMutex
	conn net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
}

//
//
// ctor/dtor
//

func newNode(registry *registry.Element, manifest *manifest.Manifest) *node {

	n := &node{
		registry: registry,
		manifest: manifest,
		bus: nil,
	}

	n.manifest.Verify()
	n.manifest.Load()

	return n
}

func (n *node) close() {}

func (n *node) setBus(bus ibus) {
	n.bus = bus
	n.bus.Register(n)
}

//
//
// inode
//

func (n *node) handleConnection(conn net.Conn, reader *bufio.Reader, writer *bufio.Writer) *error.Error {

	err := n.checkManifestForFailure()
	if err != nil {
		return err
	}

	n.manifest.Verify()
	err = n.checkManifestForFailure()
	if err != nil {
		return err
	}

	n.manifest.Load()
	err = n.checkManifestForFailure()
	if err != nil {
		return err
	}

	n.bus.Register(n)

	if n.manifest.Status.Get() != status.Verified {
		return error.New(
			error.HandshakeDenial,
			error.Spore,
			"unknown manifest verification failure",
			out.Pair("node", n.registry.ID),
			out.Pair("manifest_status", n.manifest.Status.String()))
	}

	b := newBinary(n.registry)
	err = n.checkBinaryForFailure(b)
	if err != nil {
		return err
	}

	b.verify(conn)
	err = n.checkBinaryForFailure(b)
	if err != nil {
		return err
	}

	if b.status.Get() != status.Verified {
		return error.New(
			error.HandshakeDenial,
			error.Spore,
			"unknown binary verification failure",
			out.Pair("node", n.registry.ID),
			out.Pair("binary_status", b.status.String()))
	}

	writer.WriteString("OK\n")
	writer.Flush()

	n.mu.Lock()
	defer n.mu.Unlock()
	n.conn = conn
	n.reader = reader
	n.writer = writer

	go n.listen()
	return nil
}

func (n *node) listen() {

	for {
		raw, err := n.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF || errors.Is(err, net.ErrClosed) {
				slog.Info("Disconnecting", "node", n.registry.ID)
				break
			} else {
				slog.Warn("Failure to read message", "node", n.registry.ID)
				continue
			}
		}
		raw = strings.TrimSpace(raw)
		// i := index
		// index++
		
		// witnessing starts with witness
		if strings.HasPrefix(raw, "witness") {
			body := strings.TrimPrefix(raw, "witness ")
			n.bus.WitnessNode(body, n.registry.ID)
			continue

		// publishing starts with publish
		} else if strings.HasPrefix(raw, "publish") {

			slog.Info("PUBLISH")

		// response starts with handle
		} else if strings.HasPrefix(raw, "~") {

			slog.Info("RESPONSE")

		// fallback to request
		} else {

			slog.Info("REQUEST")

		}
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	n.reader = nil
	n.writer = nil
	n.conn.Close()
}

//
//
// inode
//

func (n *node) Id() string {
	return n.registry.ID
}

func (n *node) IsConnected() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.conn != nil
}

func (n *node) IsWitness() bool {
	if n.manifest == nil {
		return false
	}
	return n.manifest.Witness
}

func (n *node) Receive(message message.Message) *error.Error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	println(message.Get())

	if n.writer == nil {
		return error.New(
			error.Generic,
			error.Node,
			"node not connected",
			out.Pair("node", n.registry.ID))
	}

	println(n.registry.ID)

	n.writer.WriteString(message.Get() + "\n")
	n.writer.Flush()
	return nil
}

//
//
// private
//

func (n *node) checkManifestForFailure() *error.Error {
	s := n.manifest.Status.Get()
	switch s {
	case status.Missing:
		return error.New(
			error.HandshakeDenial,
			error.Spore,
			"missing manifest",
			out.Pair("node", n.registry.ID))
	case status.FailedChecksum:
		return error.New(
			error.HandshakeDenial,
			error.Spore,
			"invalid manifest",
			out.Pair("node", n.registry.ID),
			out.Pair("expected_checksum", n.manifest.ExpectedChecksum))
	}
	return nil
}

func (n *node) checkBinaryForFailure(b *binary) *error.Error {
	s := b.status.Get()
	switch s {
	case status.Missing:
		return error.New(
			error.HandshakeDenial,
			error.Spore,
			"missing binary",
			out.Pair("node", n.registry.ID),
			out.Pair("path", b.path))
	case status.FailedChecksum:
		return error.New(
			error.HandshakeDenial,
			error.Spore,
			"failed checksum",
			out.Pair("node", n.registry.ID),
			out.Pair("expected_checksum", b.expectedChecksum))
	}
	return nil
}
