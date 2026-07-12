package nodes

import (
	"bufio"
	"log/slog"
	"net"
	"spored/internal/utilities/error"
)

type node struct {
	registry *registryElement
	manifest *manifest
}

//
//
// ctor/dtor
//

func newNode(registry *registryElement) inode {
	slog.Debug("Creating new node", "name", registry.Name, "manifest", registry.Manifest, "binary", registry.Binary)

	n := &node{
		registry: registry,
		manifest: newManifest(registry),
	}

	n.manifest.verify()
	n.manifest.load()

	return n
}

func (n *node) close() {}

//
//
// inode
//

func (n *node) id() string { return n.registry.Name }

func (n *node) handleConnection(conn net.Conn, reader *bufio.Reader, writer *bufio.Writer) *error.Error {
	slog.Debug("Handling connection for node", "name", n.registry.Name, "manifest", n.registry.Manifest, "binary", n.registry.Binary)

	err := n.checkManifestForFailure()
	if err != nil {
		return err
	}

	n.manifest.verify()
	err = n.checkManifestForFailure()
	if err != nil {
		return err
	}

	n.manifest.load()
	err = n.checkManifestForFailure()
	if err != nil {
		return err
	}

	if n.manifest.status.get() != Verified {
		return error.New(error.HandshakeDenial, "unknown manifest verification failure")
	}

	b := newBinary(n.registry)
	err = checkBinaryForFailure(b)
	if err != nil {
		return err
	}

	b.verify(conn)
	err = checkBinaryForFailure(b)
	if err != nil {
		return err
	}

	if b.status.get() != Verified {
		return error.New(error.HandshakeDenial, "unknown binary verification failure")
	}

	return nil
}

//
//
// private
//

func (n *node) checkManifestForFailure() *error.Error {
	s := n.manifest.status.get()
	switch s {
	case Missing:
		return error.New(error.HandshakeDenial, "missing manifest")
	case FailedChecksum:
		return error.New(error.HandshakeDenial, "invalid manifest")
	}
	return nil
}

func checkBinaryForFailure(b *binary) *error.Error {
	s := b.status.get()
	switch s {
	case Missing:
		return error.New(error.HandshakeDenial, "missing binary")
	case FailedChecksum:
		return error.New(error.HandshakeDenial, "invalid binary")
	}
	return nil
}

