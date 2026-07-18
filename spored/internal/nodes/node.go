package nodes

import (
	"bufio"
	"errors"
	"io"
	"log/slog"
	"net"
	"spored/internal/manifest"
	"spored/internal/registry"
	"spored/internal/utilities/error"
	"spored/internal/utilities/status"
	"strings"
	"sync"
)

type node struct {
	registry *registry.Element
	manifest *manifest.Manifest

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
	slog.Debug("Creating new node", "name", registry.Name, "manifest", registry.Manifest, "binary", registry.Binary)

	n := &node{
		registry: registry,
		manifest: manifest,
	}

	n.manifest.Verify()
	n.manifest.Load()

	return n
}

func (n *node) close() {}

//
//
// inode
//

func (n *node) handleConnection(conn net.Conn, reader *bufio.Reader, writer *bufio.Writer) *error.Error {
	slog.Debug("Handling connection for node", "name", n.registry.Name, "manifest", n.registry.Manifest, "binary", n.registry.Binary)

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

	if n.manifest.Status.Get() != status.Verified {
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

	if b.status.Get() != status.Verified {
		return error.New(error.HandshakeDenial, "unknown binary verification failure")
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	n.conn = conn
	n.reader = reader
	n.writer = writer
	go n.listen()

	return nil
}

func (n *node) listen() {
	slog.Info("Connecting", node, n.registry.ID)

	var index int64 = 0
	for {
		raw, err := c.reader.ReadString('\n')
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
		i := index
		index++

		// witnessing starts with witness
		if strings.HasPrefix(raw, "witness") {


		// publishing starts with publish
		} else if strings.HasPrefix(raw, "publish") {


		// response starts with handle
		} else if strings.HasPrefix(raw, "~") {


		// fallback to request
		} else {

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
// private
//

func (n *node) checkManifestForFailure() *error.Error {
	s := n.manifest.Status.Get()
	switch s {
	case status.Missing:
		return error.New(error.HandshakeDenial, "missing manifest")
	case status.FailedChecksum:
		return error.New(error.HandshakeDenial, "invalid manifest")
	}
	return nil
}

func checkBinaryForFailure(b *binary) *error.Error {
	s := b.status.Get()
	switch s {
	case status.Missing:
		return error.New(error.HandshakeDenial, "missing binary")
	case status.FailedChecksum:
		return error.New(error.HandshakeDenial, "invalid binary")
	}
	return nil
}
