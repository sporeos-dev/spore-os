package hyphae

import (
	"fmt"
	"path/filepath"
	"spored/internal/manifest"
	"spored/internal/message"
	"spored/internal/registry"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"spored/internal/utilities/status"
	"time"

	"gopkg.in/yaml.v3"
)

type Hyphae struct {
	index   int
	pending pending
	bus     ibus
	nodes   inodes
	spore   ispore
}

func New() *Hyphae {
	return &Hyphae{
		index: 0,
		pending: pending{
			channels: make(map[string]chan message.Message),
		},
	}
}

func (h *Hyphae) Set(bus ibus, nodes inodes, spore ispore) {
	h.bus = bus
	h.nodes = nodes
	h.spore = spore

	h.bus.Register(h)
}

func (h *Hyphae) Close() {
	h.bus.Unregister(h)
}

func (h *Hyphae) PrepareForInstallation(path string) (*manifest.Manifest, *registry.Element, *error.Error) {

	manifestContent, err := h.manifestRead(path)
	if err != nil {
		return nil, nil, err
	}

	manifest := &manifest.Manifest{
		Status: status.New(),
		Path: path,
		ExpectedChecksum: "not yet",
	}

	erro := yaml.Unmarshal([]byte(manifestContent), &manifest)
	if erro != nil {
		return nil, nil, error.New(
			error.Malformed,
			error.Hyphae,
			"failed to parse manifest",
			out.Pair("error", erro.Error()))
	}

	fullBinaryPath := filepath.Join(filepath.Dir(path), manifest.App)
	registry := &registry.Element{
		ID: manifest.ID,
		Name: manifest.Name,
		Manifest: path,
		Checksum: "not yet",
		Binary: fullBinaryPath,
		BinaryChecksum: "not yet",
	}

	manifestChecksum, err := h.fileHash(path)
	if err != nil {
		return nil, nil, err
	}
	manifest.ExpectedChecksum = manifestChecksum
	registry.Checksum = manifestChecksum

	binaryChecksum, err := h.fileHash(manifest.App)
	if err != nil {
		return nil, nil, err
	}
	registry.BinaryChecksum = binaryChecksum

	return manifest, registry, nil
}

//
//
// private
//

func (h *Hyphae) manifestRead(path string) (string, *error.Error) {
	handle := h.handle()
	raw := fmt.Sprintf("HYPHAE.manifest.read path=%s %s", path, handle)
	h.bus.WitnessSpore(raw)
	msg, err := message.Request(raw, h.Id())
	if err != nil {
		return "", err
	}
	ch := h.await(handle)
	err = h.bus.Request(msg)
	if err != nil {
		h.pending.mu.Lock()
		delete(h.pending.channels, handle)
		h.pending.mu.Unlock()
		return "", err
	}
	response, err := h.waitFor(handle, ch)
	if err != nil {
		return "", err
	}
	_ = response
	return "", nil
}

func (h *Hyphae) binaryHash(pid int) (string, *error.Error) {
	handle := h.handle()
	raw := fmt.Sprintf("HYPHAE.binary.hash pid=%d %s", pid, handle)
	h.bus.WitnessSpore(raw)
	msg, err := message.Request(raw, h.Id())
	if err != nil {
		return "", err
	}
	ch := h.await(handle)
	err = h.bus.Request(msg)
	if err != nil {
		h.pending.mu.Lock()
		delete(h.pending.channels, handle)
		h.pending.mu.Unlock()
		return "", err
	}
	response, err := h.waitFor(handle, ch)
	if err != nil {
		return "", err
	}
	content, err := response.Arg("content")
	if err != nil {
		return "", err
	}
	return content, nil
}

func (h *Hyphae) fileHash(path string) (string, *error.Error) {
	handle := h.handle()
	raw := fmt.Sprintf("HYPHAE.file.hash path=%s %s", path, handle)
	h.bus.WitnessSpore(raw)
	msg, err := message.Request(raw, h.Id())
	if err != nil {
		return "", err
	}
	ch := h.await(handle)
	err = h.bus.Request(msg)
	if err != nil {
		h.pending.mu.Lock()
		delete(h.pending.channels, handle)
		h.pending.mu.Unlock()
		return "", err
	}
	response, err := h.waitFor(handle, ch)
	if err != nil {
		return "", err
	}
	hash, err := response.Arg("hash")
	if err != nil {
		return "", err
	}
	return hash, nil
}

func (h *Hyphae) nodeSpawn(path string) *error.Error {
	handle := h.handle()
	raw := fmt.Sprintf("HYPHAE.node.spawn path=%s %s", path, handle)
	h.bus.WitnessSpore(raw)
	msg, err := message.Request(raw, h.Id())
	if err != nil {
		return err
	}
	ch := h.await(handle)
	err = h.bus.Request(msg)
	if err != nil {
		h.pending.mu.Lock()
		delete(h.pending.channels, handle)
		h.pending.mu.Unlock()
		return err
	}
	_, err = h.waitFor(handle, ch)
	return nil
}

func (h *Hyphae) nodeKill(pid int) *error.Error {
	handle := h.handle()
	raw := fmt.Sprintf("HYPHAE.node.kill pid=%d %s", pid, handle)
	h.bus.WitnessSpore(raw)
	msg, err := message.Request(raw, h.Id())
	if err != nil {
		return err
	}
	ch := h.await(handle)
	err = h.bus.Request(msg)
	if err != nil {
		h.pending.mu.Lock()
		delete(h.pending.channels, handle)
		h.pending.mu.Unlock()
		return err
	}
	_, err = h.waitFor(handle, ch)
	return err
}

func (h *Hyphae) handle() string {
	h.index++
	return fmt.Sprintf("~hyphae-%d", h.index)
}

// await registers a channel for the given handle and returns it.
// Call this before sending the request, then block on the returned channel.
func (h *Hyphae) await(handle string) chan message.Message {
	ch := make(chan message.Message, 1)
	h.pending.mu.Lock()
	h.pending.channels[handle] = ch
	h.pending.mu.Unlock()
	return ch
}

// waitFor blocks until a response arrives on ch or 5 seconds elapse.
func (h *Hyphae) waitFor(handle string, ch chan message.Message) (message.Message, *error.Error) {
	select {
	case msg := <-ch:
		return msg, nil
	case <-time.After(5 * time.Second):
		h.pending.mu.Lock()
		delete(h.pending.channels, handle)
		h.pending.mu.Unlock()
		return nil, error.New(error.Timeout, error.Hyphae, "timed out waiting for response", out.Pair("handle", handle))
	}
}

//
//
// INode
//

func (h *Hyphae) Id() string {
	return "hyphae-in-spore"
}

func (h *Hyphae) IsConnected() bool {
	return true
}

func (h *Hyphae) IsWitness() bool {
	return false
}

func (h *Hyphae) GetManifest() *manifest.Manifest {
	return nil
}

func (h *Hyphae) Receive(msg message.Message) *error.Error {
	handle := msg.Handle()
	h.pending.mu.Lock()
	ch, ok := h.pending.channels[handle]
	if ok {
		delete(h.pending.channels, handle)
	}
	h.pending.mu.Unlock()

	if ok {
		ch <- msg
	}
	return nil
}
