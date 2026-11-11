// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package node

import (
	"bufio"
	"errors"
	"io"
	"net"
	"spored/internal/event"
	message_event "spored/internal/event/message"
	"spored/internal/hyphae"
	"spored/internal/iface"
	"spored/internal/manifest"
	"spored/internal/nodes/broadcast_node"
	"spored/internal/nodes/request_node"
	"spored/internal/nodes/response_node"
	"spored/internal/nodes/witness_node"
	"spored/internal/pal"
	"spored/internal/registry"
	"spored/internal/utilities/await"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"spored/internal/utilities/status"
	"spored/internal/witness"
	"strings"
	"sync"
	"time"
)

type Single struct {
	registry  *registry.Element
	manifest  *manifest.Manifest

	mu     sync.RWMutex
	conn   net.Conn
	reader *bufio.Reader
	writer *writer

	verificationDone   chan struct{}
	verificationPassed bool

	pending *await.Pending

	// events
	broadcastHandler *broadcast_node.BroadcastHandler
	requestHandler *request_node.RequestHandler
	responseHandler *response_node.ResponseHandler
	witnessHandler *witness_node.WitnessHandler
}

type State struct {
	Connected bool
	Pid       int
}

//
//
// ctor/dtor
//

func New(registry *registry.Element) *Single {

	n := &Single{
		registry: registry,
		manifest: nil,
		pending:  await.New(error.Node).WithTimeout(time.Second * 5),

		broadcastHandler: broadcast_node.New(),
		requestHandler: request_node.New(),
		responseHandler: response_node.New(),
		witnessHandler: witness_node.New(),
	}

	n.manifest.Load()
	n.manifest.Verify()

	return n
}

//
//
// inode
//

func (s *Single) HandleConnection(conn net.Conn, reader *bufio.Reader, wr *bufio.Writer) {

	writer := newWriter(conn, wr, s.registry.ID)

	err := s.checkManifestForFailure()
	if err != nil {
		witness.Error(error.HandshakeDenial, error.Node, "manifest failure")
		wr.WriteString("FAIL manifest failure\n")
		wr.Flush()
		conn.Close()
		return
	}

	s.manifest.Verify()
	if s.manifest.Status.Get() == status.RequiresDeveloper {

		if s.registry.Checksum == string(manifest.Developer) {
			s.manifest.Status.Set(status.Verified)
		} else {
			witness.Error(error.HandshakeDenial, error.Node, "developer checksum mismatch")
			wr.WriteString("FAIL developer checksum mismatch\n")
			wr.Flush()
			conn.Close()
			return
		}

	} else if s.manifest.Status.Get() == status.RequiresUserSpace {

		checksum, ok := hyphae.FileHash(s.manifest.Path)
		if !ok {
			witness.Error(error.HandshakeDenial, error.Node, "hash error")
			wr.WriteString("FAIL hash error\n")
			wr.Flush()
			conn.Close()
			return
		}
		if checksum == s.manifest.ExpectedChecksum {
			s.manifest.Status.Set(status.Verified)
		} else {
			s.manifest.Status.Set(status.FailedChecksum)
			witness.Error(error.HandshakeDenial, error.Node, "failed manifest verification")
			wr.WriteString("FAIL manifest verification\n")
			wr.Flush()
			conn.Close()
			return
		}
	}

	err = s.checkManifestForFailure()
	if err != nil {
		witness.Error(error.HandshakeDenial, error.Node, "manifest failure")
		wr.WriteString("FAIL manifest failure\n")
		wr.Flush()
		conn.Close()
		return
	}

	s.manifest.Load()
	err = s.checkManifestForFailure()
	if err != nil {
		witness.Error(error.HandshakeDenial, error.Node, "manifest failure")
		wr.WriteString("FAIL manifest failure\n")
		wr.Flush()
		conn.Close()
		return
	}

	if s.manifest.Status.Get() != status.Verified {
		witness.Error(error.HandshakeDenial, error.Node, "manifest not verified")
		wr.WriteString("FAIL manifest not verified\n")
		wr.Flush()
		conn.Close()
		return
	}

	var b *binary
	if s.manifest.Trust == manifest.Developer {
		b = newDeveloperBinary(s.registry)
	} else {
		b = newBinary(s.registry)
	}
	err = s.checkBinaryForFailure(b)
	if err != nil {
		witness.Error(error.HandshakeDenial, error.Node, "binary failure")
		wr.WriteString("FAIL binary failure\n")
		wr.Flush()
		conn.Close()
		return
	}

	connectionActivated := false
	b.verify(conn)
	if b.status.Get() == status.RequiresUserSpace {
		writer.WriteRaw("OK")
		s.activateConnection(conn, reader, writer, true)
		connectionActivated = true
		defer s.finishVerification(false)

		pid, erro := pal.ProcessID(conn)
		if erro != nil {
			witness.Error(error.HandshakeDenial, error.Node, "failed to get pid")
			wr.WriteString("FAIL failed to get pid\n")
			wr.Flush()
			conn.Close()
			return
		}

		checksum, err := hyphae.HashBinary(pid)
		if err != nil {
			witness.Error(error.HandshakeDenial, error.Node, "failed binary hash")
			wr.WriteString("FAIL binary hash\n")
			wr.Flush()
			conn.Close()
			return
		}
		if checksum == s.registry.BinaryChecksum {
			b.status.Set(status.Verified)
		} else {
			b.status.Set(status.FailedChecksum)
			witness.Error(error.HandshakeDenial, error.Node, "failed manifest verification")
			wr.WriteString("FAIL manifest verification\n")
			wr.Flush()
			conn.Close()
			return
		}
	}
	err = s.checkBinaryForFailure(b)
	if err != nil {
		witness.Error(error.HandshakeDenial, error.Node, "binary failure")
		wr.WriteString("FAIL binary failure\n")
		wr.Flush()
		conn.Close()
		return
	}

	if b.status.Get() != status.Verified {
		witness.Error(error.HandshakeDenial, error.Node, "unknown binary verification failure")
		wr.WriteString("FAIL binary verification\n")
		wr.Flush()
		conn.Close()
		return
	}

	if !connectionActivated {
		writer.WriteRaw("OK")
		s.activateConnection(conn, reader, writer, false)
	}

	// The node may service the temporary binary.hash request above, but does
	// not receive permissions until its binary checksum has been verified.
	// s.permitter.RegisterNode(s.manifest)
	if connectionActivated {
		s.finishVerification(true)
	}

	// register parts to the event system
	s.broadcastHandler.Open(s.manifest, s.writer)
	s.requestHandler.Open(s.manifest, s.writer)
	s.responseHandler.Open(s.manifest, s.writer)
	s.witnessHandler.Open(s.manifest, s.writer)

	witness.Spore("handshake successful", out.Pair("node", s.registry.ID))

	err = s.pending.Resolve("lazy")
	if err != nil {
		witness.Error(error.HandshakeDenial, error.Node, "failed to resolve pending")
		wr.WriteString("FAIL resolve pending\n")
		wr.Flush()
		conn.Close()
		return
	}
}

func (s *Single) activateConnection(conn net.Conn, reader *bufio.Reader, writer *writer, provisional bool) {
	_ = conn.SetReadDeadline(time.Time{})

	s.mu.Lock()
	s.conn = conn
	s.reader = reader
	s.writer = writer
	if provisional {
		s.verificationDone = make(chan struct{})
		s.verificationPassed = false
	}
	s.mu.Unlock()

	go s.listen()
}

func (s *Single) finishVerification(passed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.verificationDone == nil {
		return
	}
	s.verificationPassed = passed
	close(s.verificationDone)
	s.verificationDone = nil
}

func (s *Single) waitForVerification() bool {
	s.mu.RLock()
	done := s.verificationDone
	s.mu.RUnlock()
	if done == nil {
		return true
	}
	<-done
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.verificationPassed
}

// readStatus reports the outcome of readMessage without spelling the builtin
// error interface, which is shadowed in this file by the utilities/error import.
type readStatus int

type readError interface {
	Error() string
}

const (
	readOK readStatus = iota
	readClosed
	readFailure
)

// readMessage reads one complete wire message, accumulating physical lines
// that make up a block value (key=<< ... >>), mirroring spore_c's Client::listen().
func (s *Single) readMessage() (string, readStatus, readError) {
	var line string
	multiline := false

	for {
		raw, readErr := s.reader.ReadString('\n')
		if readErr != nil {
			if readErr == io.EOF || errors.Is(readErr, net.ErrClosed) {
				return "", readClosed, readErr
			}
			return "", readFailure, readErr
		}

		currentLine := strings.TrimSuffix(raw, "\n")
		currentLine = strings.TrimSuffix(currentLine, "\r")

		if !multiline {
			if strings.Contains(currentLine, "<<") {
				multiline = true
				line = currentLine + "\n"
				continue
			}
			line = currentLine
		} else {
			isClose := currentLine == ">>" || strings.HasPrefix(currentLine, ">> ")
			if !isClose {
				line += currentLine + "\n"
				continue
			}
			line += currentLine
			if strings.Contains(currentLine, "<<") {
				continue
			}
		}

		multiline = false
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		return line, readOK, nil
	}
}

func (s *Single) listen() {

listen:
	for {
		raw, status, readErr := s.readMessage()
		switch status {
		case readClosed:
			witness.Spore("node disconnecting", out.Pair("node", s.registry.ID))
			break listen
		case readFailure:
			if pal.IsTerminalPeerReadError(readErr) {
				break listen
			}
			s.Receive(
				error.New(
					error.InitializationFailure,
					error.Node,
					"read failure",
					out.Pair("node", s.registry.ID),
					out.Pair("error", readErr.Error())))
			if !isRetryableReadError(readErr) {
				break listen
			}
			continue
		}

		if !strings.HasPrefix(raw, "~") && !strings.Contains(strings.Fields(raw)[0], ".binary.hash") {
			if !s.waitForVerification() {
				continue
			}
		}

		msg := message_event.Raw(raw, s.registry.ID)
		event.Dispatch().Message.Notify(msg)
	}

	// close handlers
	s.broadcastHandler.Close()
	s.requestHandler.Close()
	s.responseHandler.Close()
	s.witnessHandler.Close()

	s.mu.Lock()
	defer s.mu.Unlock()
	s.reader = nil
	s.writer = nil
	s.conn.Close()
	s.conn = nil
}

func (s *Single) returnRoutingError(err *error.Error, msg iface.Message) {
	if receiveErr := s.Receive(err.WithMessage(msg)); receiveErr != nil {
		witness.Error(
			error.Generic,
			error.Node,
			"failed to return routing error",
			out.Pair("node", s.registry.ID),
			out.Pair("error", receiveErr.Error()))
	}
}

func isRetryableReadError(readErr readError) bool {
	netErr, ok := readErr.(net.Error)
	return ok && netErr.Timeout()
}

func (s *Single) ProcessID() (int, bool) {
	pid, err := pal.ProcessID(s.conn)
	if err != nil {
		return 0, false
	}
	return pid, true
}

//
//
// private
//

func (s *Single) checkManifestForFailure() *error.Error {
	st := s.manifest.Status.Get()
	switch st {
	case status.Missing:
		return error.New(
			error.HandshakeDenial,
			error.Spore,
			"missing manifest",
			out.Pair("node", s.registry.ID))
	case status.FailedChecksum:
		return error.New(
			error.HandshakeDenial,
			error.Spore,
			"invalid manifest",
			out.Pair("node", s.registry.ID),
			out.Pair("expected_checksum", s.manifest.ExpectedChecksum))
	case status.Malformed:
		return error.New(
			error.HandshakeDenial,
			error.Spore,
			"malformed manifest",
			out.Pair("node", s.registry.ID))
	}
	return nil
}

func (s *Single) checkBinaryForFailure(b *binary) *error.Error {
	st := b.status.Get()
	switch st {
	case status.Missing:
		return error.New(
			error.HandshakeDenial,
			error.Spore,
			"missing binary",
			out.Pair("node", s.registry.ID),
			out.Pair("path", b.path))
	case status.FailedChecksum:
		return error.New(
			error.HandshakeDenial,
			error.Spore,
			"failed checksum",
			out.Pair("node", s.registry.ID),
			out.Pair("expected_checksum", b.expectedChecksum))
	case status.Malformed:
		return error.New(
			error.HandshakeDenial,
			error.Spore,
			"malformed (canary)",
			out.Pair("node", s.registry.ID))
	}
	return nil
}
