package hyphae

import (
	"fmt"
	"path/filepath"
	"spored/internal/iface"
	"spored/internal/manifest"
	"spored/internal/message"
	"spored/internal/registry"
	"spored/internal/utilities/await"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"spored/internal/utilities/status"
	"spored/internal/witness"
	"time"

	"gopkg.in/yaml.v3"
)

type Hyphae struct {
	index   int
	pending *await.Pending
	bus     ibus
	nodes   inodes
	permissions ipermissions
	spore   ispore
}

func New() *Hyphae {
	return &Hyphae{
		index: 0,
		pending: await.New(error.Hyphae).WithTimeout(time.Second * 2),
	}
}

func (h *Hyphae) Set(bus ibus, nodes inodes, permissions ipermissions,spore ispore) {
	h.bus = bus
	h.nodes = nodes
	h.permissions = permissions
	h.spore = spore

	h.bus.Register(h)
}

func (h *Hyphae) Close() {
	h.bus.Unregister(h)
}

//
//
// public
// hyphae
//

func (h *Hyphae) ManifestRead(path string) (string, *error.Error) {
	return h.manifestRead(path)
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

	binaryChecksum, err := h.fileHash(fullBinaryPath)
	if err != nil {
		return nil, nil, err
	}
	registry.BinaryChecksum = binaryChecksum

	return manifest, registry, nil
}

func (h *Hyphae) HashFile(path string) (string, *error.Error) {
	return h.fileHash(path)
}

func (h *Hyphae) Spawn(path string) *error.Error {
	return h.nodeSpawn(path)
}

func (h *Hyphae) Kill(pid int) *error.Error {
	return h.nodeKill(pid)
}

//
//
// private
// hyphae calls
//

func (h *Hyphae) manifestRead(path string) (string, *error.Error) {
	handle := h.handle()
	raw := fmt.Sprintf(`HYPHAE.manifest.read path="%s" ~%s`, path, handle)
	msg, ok := message.Request(raw, h.Id())
	if !ok {
		return "", error.New(
			error.Malformed,
			error.Hyphae,
			"failure to read manifest")
	}
	witness.Send(msg)

	ch := h.pending.Await(handle)
	err := h.bus.Request(msg)
	if err != nil {
		h.pending.Delete(handle)
		return "", err
	}
	response, err := h.pending.WaitFor(handle, ch)
	if err != nil {
		return "", err
	}
	if response.Flag("error") {
		return "", error.New(
			error.Generic, 
			error.Hyphae, 
			response.ArgIf("what", "manifest read failed"))
	}

	content, ok := response.Arg("content")
	if !ok {
		return "", error.New(
			error.Missing,
			error.Hyphae,
			"missing in response",
			out.Pair("argument", "content"))
	}
	return content, nil
}

func (h *Hyphae) binaryHash(pid int) (string, *error.Error) {
	handle := h.handle()
	raw := fmt.Sprintf(`HYPHAE.binary.hash pid=%d ~%s`, pid, handle)
	msg, ok := message.Request(raw, h.Id())
		if !ok {
		return "", error.New(
			error.Malformed,
			error.Hyphae,
			"failure to hash binary")
	}
	witness.Send(msg)

	ch := h.pending.Await(handle)
	err := h.bus.Request(msg)
	if err != nil {
		h.pending.Delete(handle)
		return "", err
	}
	response, err := h.pending.WaitFor(handle, ch)
	if err != nil {
		return "", err
	}
	if response.Flag("error") {
		return "", error.New(
			error.Generic, 
			error.Hyphae, 
			response.ArgIf("what", "binary hash failed"))
	}

	content, ok := response.Arg("content")
	if !ok {
		return "", error.New(
			error.Missing,
			error.Hyphae,
			"missing in response",
			out.Pair("argument", "content"))
	}
	return content, nil
}

func (h *Hyphae) fileHash(path string) (string, *error.Error) {
	handle := h.handle()
	raw := fmt.Sprintf(`HYPHAE.file.hash path=%s ~%s`, path, handle)
	msg, ok := message.Request(raw, h.Id())
		if !ok {
		return "", error.New(
			error.Malformed,
			error.Hyphae,
			"failure to hash file")
	}
	witness.Send(msg)

	ch := h.pending.Await(handle)
	err := h.bus.Request(msg)
	if err != nil {
		h.pending.Delete(handle)
		return "", err
	}
	response, err := h.pending.WaitFor(handle, ch)
	if err != nil {
		return "", err
	}
	if response.Flag("error") {
		return "", error.New(
			error.Generic, 
			error.Hyphae, 
			response.ArgIf("what", "file hash failed"))
	}

	hash, ok := response.Arg("hash")
	if !ok {
		return "", error.New(
			error.Missing,
			error.Hyphae,
			"missing in response",
			out.Pair("argument", "hash"))
	}
	return hash, nil
}

func (h *Hyphae) nodeSpawn(path string) *error.Error {
	handle := h.handle()
	raw := fmt.Sprintf(`HYPHAE.node.spawn binary="%s" ~%s`, path, handle)
	msg, ok := message.Request(raw, h.Id())
	if !ok {
		return error.New(
			error.Malformed,
			error.Hyphae,
			"failure to spawn node")
	}
	witness.Send(msg)

	ch := h.pending.Await(handle)
	err := h.bus.Request(msg)
	if err != nil {
		h.pending.Delete(handle)
		return err
	}
	response, err := h.pending.WaitFor(handle, ch)
	if err != nil {
		return err
	}
	if response.Flag("error") {
		return error.New(
			error.Generic, 
			error.Hyphae, 
			response.ArgIf("what", "node spawn failed"))
	}

	return nil
}

func (h *Hyphae) nodeKill(pid int) *error.Error {
	handle := h.handle()
	raw := fmt.Sprintf("HYPHAE.node.kill pid=%d ~%s", pid, handle)
	msg, ok := message.Request(raw, h.Id())
		if !ok {
		return error.New(
			error.Malformed,
			error.Hyphae,
			"failure to kill node")
	}
	witness.Send(msg)

	ch := h.pending.Await(handle)
	err := h.bus.Request(msg)
	if err != nil {
		h.pending.Delete(handle)
		return err
	}
	response, err := h.pending.WaitFor(handle, ch)
	if err != nil {
		return err
	}
	if response.Flag("error") {
		return error.New(
			error.Generic, 
			error.Hyphae, 
			response.ArgIf("what", "node kill failed"))
	}

	return nil
}

//
//
// private
// hyphae call
// helpers
//

func (h *Hyphae) handle() string {
	h.index++
	return fmt.Sprintf("hyphae-%d", h.index)
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

func (h *Hyphae) Receive(msg iface.Message) *error.Error {
	return h.pending.Receive(msg)
}

func (h *Hyphae) Witness(msg iface.Message) {}
