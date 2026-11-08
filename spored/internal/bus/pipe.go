// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package bus

import (
	"fmt"
	"slices"
	"spored/internal/cparser"
	"spored/internal/iface"
	"spored/internal/manifest"
	"spored/internal/message"
	"spored/internal/utilities/await"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	witnesslog "spored/internal/witness"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type pipe struct {
	pending *await.Pending
	bus     *Bus
}

type ResponsePipe interface {
	Args() map[string]string
	Flags() []string
}

func newPipe() *pipe {
	return &pipe{
		pending: await.New(error.Pipe).WithTimeout(time.Second * 10),
		bus:     nil,
	}
}

func (p *pipe) setBus(bus *Bus) {
	p.bus = bus
}

func (p *pipe) close() {}

// should be run on a goroutine
func (p *pipe) pipe(msg iface.Message, node INode) {

	complete, _ := msg.Arg("raw")
	cast := node.Id()
	origin := msg.Origin()
	args := make(map[string]string)
	flags := make([]string, 0)
	callerHandle := pipeCallerHandle(msg)

	index := 0
	for {
		raw, ok := msg.Arg(fmt.Sprint(index))
		if !ok {
			// unexpected failures
			switch raw {
			case "bad-key":
				node.Receive(error.New(
					error.Malformed,
					error.Pipe,
					"bad key in pipe"))
				return
			case "low-index":
				node.Receive(error.New(
					error.Malformed,
					error.Pipe,
					"low index in pipe"))
				return

				// finalizer case
			case "high-index":
				return
			}
		}
		index++
		raw = strings.TrimSpace(raw)

		//
		// error
		// handling
		//

		// should not be empty
		if raw == "" {
			node.Receive(error.New(
				error.Empty,
				error.Pipe,
				"empty pipe segment",
				out.Pair("index", fmt.Sprint(index)),
				out.Pair("raw", raw),
				out.Pair("complete", complete)))
			return

			// should not be pipe within a pipe
		} else if message.IsPipe(raw) {
			node.Receive(error.New(
				error.Malformed,
				error.Pipe,
				"nested pipe parsing failure",
				out.Pair("index", fmt.Sprint(index)),
				out.Pair("raw", raw),
				out.Pair("complete", complete)))
			return

			// should not be a response
		} else if strings.HasPrefix(raw, "~") {
			node.Receive(error.New(
				error.Malformed,
				error.Pipe,
				"unexpected response in pipe",
				out.Pair("index", fmt.Sprint(index)),
				out.Pair("raw", raw),
				out.Pair("complete", complete)))
			return
		}

		//
		// applying
		//

		pm, ok := cparser.Parse(raw)
		if !ok {
			node.Receive(error.New(
				error.Malformed,
				error.Pipe,
				"failed to parse witness message",
				out.Pair("index", fmt.Sprint(index)),
				out.Pair("raw", raw),
				out.Pair("complete", complete)))
			return
		}
		for key, value := range args {
			pm.Args[key] = value
		}
		for _, flag := range flags {
			if !slices.Contains(pm.Flags, flag) {
				pm.Flags = append(pm.Flags, flag)
			}
		}

		// peek ahead: the last stage responds directly to the original caller
		_, hasNext := msg.Arg(fmt.Sprint(index))
		isLast := !hasNext

		//
		// routing
		//
		
		timeout, ok := handleTimeout(&pm)
		if !ok {
			node.Receive(error.New(
				error.Malformed,
				error.Pipe,
				"invalid timeout argument",
				out.Pair("index", fmt.Sprint(index)),
				out.Pair("raw", raw),
				out.Pair("complete", complete)))
			return
		}

		if !isLast {
			pm.Handle = pipeHandle()
		} else if callerHandle != "" {
			pm.Handle = callerHandle
		} else if pm.Handle == "" {
			pm.Handle = pipeHandle()
		}
		// intermediate stages route back to the pipe itself; only the
		// last stage responds directly to the original caller
		stageCast := cast
		if !isLast {
			stageCast = message.DelegatedCast(p.Id(), origin)
		}
		raw = pm.Stringify()
		witnesslog.Incoming(raw, stageCast)

		request, ok := message.Request(raw, stageCast)
		if !ok {
			node.Receive(error.New(
				error.Malformed,
				error.Pipe,
				"failed to parse request message",
				out.Pair("index", fmt.Sprint(index)),
				out.Pair("raw", raw),
				out.Pair("complete", complete)))
			return
		}

		if isLast {
			if err := p.bus.Request(request); err != nil {
				node.Receive(err)
			}
			return
		}

		ch := p.pending.Await(request.Handle())
		err := p.bus.Request(request)
		if err != nil {
			p.pending.Delete(request.Handle())
			if callerRequest, ok := pipeTimeoutRequest(msg, request.Capability(), cast); ok {
				err.WithMessage(callerRequest)
			}
			node.Receive(err)
			return
		}

		response, err := p.pending.WaitForTimeout(request.Handle(), ch, timeout)
		if err != nil {
			callerRequest, ok := pipeTimeoutRequest(msg, request.Capability(), cast)
			if !ok {
				witnesslog.Error(
					error.Missing,
					error.Pipe,
					"cannot report pipeline timeout without a caller handle",
					out.Pair("complete", complete))
				return
			}
			node.Receive(err.WithMessage(callerRequest))
			return
		}
		if response.Flag("cancelled") || response.Flag("error") {
			reply, ok := message.ResponseWithHandle(response, callerHandle)
			if !ok {
				witnesslog.Error(
					error.Malformed,
					error.Pipe,
					"cannot return pipeline response to caller",
					out.Pair("complete", complete))
				return
			}
			if err := node.Receive(reply); err != nil {
				witnesslog.Error(
					error.Generic,
					error.Pipe,
					"failed to send response",
					out.Pair("complete", complete))
			}
			return
		}
		responsePipe, ok := response.(ResponsePipe)
		if !ok {
			node.Receive(error.New(
				error.Generic,
				error.Pipe,
				"pipe response does not provide arguments and flags",
				out.Pair("index", fmt.Sprint(index)),
				out.Pair("raw", raw),
				out.Pair("complete", complete)))
			return
		}
		for key, value := range responsePipe.Args() {
			args[key] = value
		}
		for _, flag := range responsePipe.Flags() {
			if flag == "ok" || flag == "error" || flag == "cancelled" {
				continue
			}
			if !slices.Contains(flags, flag) {
				flags = append(flags, flag)
			}
		}
	}
}

//
//
// INode
//

func (p *pipe) Id() string {
	return "SPORE.pipe"
}

func (p *pipe) IsConnected() bool {
	return true
}

func (p *pipe) IsWitness() bool {
	return false
}

func (p *pipe) GetManifest() *manifest.Manifest {
	return nil
}

func (p *pipe) Receive(msg iface.Message) *error.Error {
	return p.pending.Receive(msg)
}

func (p *pipe) Witness(msg iface.Message) {}

//
//
// private internal
//

var pipeHandleIndex atomic.Int64

func pipeCallerHandle(pipe iface.Message) string {
	for index := 0; ; index++ {
		raw, ok := pipe.Arg(strconv.Itoa(index))
		if !ok {
			return ""
		}
		stage, ok := cparser.Parse(strings.TrimSpace(raw))
		if !ok || stage.Handle == "" {
			continue
		}
		return stage.Handle
	}
}

func pipeTimeoutRequest(pipe iface.Message, capability, cast string) (iface.Message, bool) {
	handle := pipeCallerHandle(pipe)
	if handle == "" {
		return nil, false
	}
	return message.Request(fmt.Sprintf("%s ~%s", capability, handle), cast)
}

func pipeHandle() string {
	return fmt.Sprintf("spore_pipe_%d", pipeHandleIndex.Add(1))
}

func handleTimeout(pm *cparser.ParsedMessage) (*time.Duration, bool) {
	raw, ok := pm.Args["timeout"]
	if !ok {
		return nil, true
	}
	delete(pm.Args, "timeout")

	if raw == "false" {
		d := time.Duration(0)
		return &d, true
	}

	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 0 {
		return nil, false
	}
	d := time.Duration(seconds) * time.Second
	return &d, true
}
