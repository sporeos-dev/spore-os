package nodes

import (
	"bufio"
	"errors"
	"io"
	"net"
	"spored/internal/cparser"
	"spored/internal/iface"
	"spored/internal/manifest"
	"spored/internal/message"
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

type spawner interface {
	Spawn(nodeid string) *error.Error
}

type permitter interface {
	Can(nodeid string, capability string) bool
}

type node struct {
	registry  *registry.Element
	manifest  *manifest.Manifest
	bus       ibus
	spawner   spawner
	permitter permitter

	mu     sync.RWMutex
	conn   net.Conn
	reader *bufio.Reader
	writer *writer

	pending *await.Pending
}

type State struct {
	Connected bool
	Pid int
}

//
//
// ctor/dtor
//

func newNode(registry *registry.Element, manifest *manifest.Manifest) *node {

	n := &node{
		registry: registry,
		manifest: manifest,
		bus:      nil,
		pending: await.New(error.Node).WithTimeout(time.Second * 5),
	}

	n.manifest.Verify()
	n.manifest.Load()

	return n
}

func (n *node) close() {
	n.bus.Unregister(n)
}

func (n *node) set(bus ibus, spawner spawner, permitter permitter) {
	n.spawner = spawner
	n.bus = bus
	n.permitter = permitter
	n.bus.Register(n)
}

func (n *node) state() *State {
	n.mu.RLock()
	defer n.mu.RUnlock()

	pid := -1
	if n.conn != nil {
		pid, err := pal.ProcessID(n.conn)
		if err == nil {
			return &State{
				Connected: true,
				Pid:       pid,
			}
		}
	}
	return &State{
		Connected: n.conn != nil,
		Pid:       pid,
	}
}

//
//
// inode
//

func (n *node) handleConnection(conn net.Conn, reader *bufio.Reader, wr *bufio.Writer, hyphae ihyphae) *error.Error {

	writer := newWriter(wr)

	err := n.checkManifestForFailure()
	if err != nil {
		return err
	}

	n.manifest.Verify()
	if n.manifest.Status.Get() == status.RequiresUserSpace {
		checksum, err := hyphae.HashFile(n.manifest.Path)
		if err != nil {
			return err
		}
		if checksum == n.manifest.ExpectedChecksum {
			n.manifest.Status.Set(status.Verified)
		} else {
			n.manifest.Status.Set(status.FailedChecksum)
			return error.New(
				error.HandshakeDenial,
				error.Node,
				"failed manifest verification",
				out.Pair("checksum", checksum),
				out.Pair("expected", n.manifest.ExpectedChecksum))
		}
	}
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
	if b.status.Get() == status.RequiresUserSpace {
		checksum, err := hyphae.HashFile(n.registry.Binary)
		if err != nil {
			return err
		}
		if checksum == n.registry.BinaryChecksum {
			b.status.Set(status.Verified)
		} else {
			b.status.Set(status.FailedChecksum)
			return error.New(
				error.HandshakeDenial,
				error.Node,
				"failed manifest verification",
				out.Pair("checksum", checksum),
				out.Pair("expected", n.registry.BinaryChecksum))
		}
	}
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

	writer.WriteRaw("OK")

	n.mu.Lock()
	n.conn = conn
	n.reader = reader
	n.writer = writer
	n.mu.Unlock()

	witness.Send(
		message.Witness(
			"handshake successful",
			out.Pair("node", n.registry.ID)))

	go n.listen()
	n.pending.Resolve("lazy")

	return nil
}

func (n *node) listen() {

	for {
		raw, err := n.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF || errors.Is(err, net.ErrClosed) {
				witness.Send(
					message.Witness(
						"node disconnecting",
						out.Pair("node", n.registry.ID)))
				break
			} else {
				n.Receive(
					error.New(
						error.InitializationFailure,
						error.Node,
						"read failure",
						out.Pair("node", n.registry.ID)))
				continue
			}
		}
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		// i := index
		// index++

		// pipe message
		if message.IsPipe(raw) {

			witness.Send(message.Incoming(raw, n.registry.ID))
			pipeMsg, ok := message.Pipe(raw, n.registry.ID)
			if !ok {
				n.Receive(error.New(
					error.Malformed,
					error.Node,
					"failed to parse pipe",
					out.Pair("node", n.registry.ID),
					out.Pair("raw", raw)))
				continue
			}

			go n.bus.Pipe(pipeMsg, n)

			// witnessing starts with witness
		} else if strings.HasPrefix(raw, "witness") {

			pm, ok := cparser.Validate(raw)
			var body string
			if ok {
				body = pm.Args["body"]
			} else {
				body = strings.TrimPrefix(raw, "witness ")
			}
			witness.Send(message.Node(body, n.registry.ID))

			// publishing starts with publish
		} else if strings.HasPrefix(raw, "publish") {

			witness.Send(message.Incoming(raw, n.registry.ID))
			broadcast, ok := message.Broadcast(raw, n.registry.ID)
			if !ok {
				n.Receive(error.New(
					error.Malformed,
					error.Node,
					"failed to publish",
					out.Pair("node", n.registry.ID),
					out.Pair("raw", raw)))
				continue
			}

			topic := broadcast.Capability()
			ok = false
			for _, el := range n.manifest.Topics {
				if topic == el.Name {
					ok = true
					break
				}
			}
			if !ok {
				n.Receive(error.New(
					error.NotPermitted,
					error.Node,
					"topic not announced in the manifest",
					out.Pair("node", n.registry.ID),
					out.Pair("topic", topic)).
					WithMessage(broadcast))
				continue
			}

			err := n.bus.Broadcast(broadcast)
			if err != nil {
				n.Receive(err)
			}

			// response starts with handle
		} else if strings.HasPrefix(raw, "~") {

			witness.Send(message.Incoming(raw, n.registry.ID))
			response, ok := message.Response(raw, n.registry.ID)
			if !ok {
				n.Receive(error.New(
					error.Malformed,
					error.Node,
					"failed to respond",
					out.Pair("node", n.registry.ID),
					out.Pair("raw", raw)))
				continue
			}
			err := n.bus.Response(response)
			if err != nil {
				n.Receive(err)
			}

			// fallback to request
		} else {

			witness.Send(message.Incoming(raw, n.registry.ID))
			request, ok := message.Request(raw, n.registry.ID)
			if !ok {
				n.Receive(error.New(
					error.Malformed,
					error.Node,
					"failed to request",
					out.Pair("node", n.registry.ID),
					out.Pair("raw", raw)))
				continue
			}

			// if n.manifest.Trust == manifest.StandardTrust || n.manifest.Trust == manifest.Untrusted {
			// 	command := msg.Command()
			// 	if !n.permitter.Can(n.registry.ID, command) {
			// 		n.Error(error.New(
			// 			error.NotPermitted,
			// 			error.Node,
			// 			"permission required",
			// 			out.Pair("node", n.registry.ID),
			// 			out.Pair("capability", command)), "", "")
			// 		return
			// 	}
			// }

			err := n.bus.Request(request)
			if err != nil {
				n.Receive(err)
			}
		}
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	n.reader = nil
	n.writer = nil
	n.conn.Close()
}

func (n *node) ProcessID() (int, bool) {
	pid, err := pal.ProcessID(n.conn)
	if err != nil {
		return 0, false
	}
	return pid, true
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

func (n *node) GetManifest() *manifest.Manifest {
	return n.manifest
}

func (n *node) Receive(message iface.Message) *error.Error {
	
	n.mu.RLock()
	writer := n.writer
	n.mu.RUnlock()

	if writer == nil {

		if n.manifest.Launch != manifest.Lazy {
			return error.New(
				error.Generic,
				error.Node,
				"node not connected",
				out.Pair("node", n.registry.ID)).
				WithMessage(message)
		}

		if n.pending.Has("lazy") {
			return error.New(
				error.Generic,
				error.Node,
				"lazy instantiation already in progress",
				out.Pair("node", n.registry.ID)).
				WithMessage(message)
		}
		ch := n.pending.Await("lazy")

		err := n.spawner.Spawn(n.registry.ID)
		if err != nil {
			n.pending.Delete("lazy")
			return err
		}

		response, err := n.pending.WaitFor("lazy", ch)
		if err != nil {
			return err
		}
		if response.Flag("error") {
			return error.New(
				error.Generic,
				error.Node,
				response.ArgIf("what", "lazy instantiation failed"))
		}
	}
	
	n.mu.RLock()
	defer n.mu.RUnlock()

	n.writer.WriteMessage(message)
	return nil
}

func (n *node) Witness(message iface.Message) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.writer == nil {
		return
	}

	n.writer.WriteWitness(message)
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
