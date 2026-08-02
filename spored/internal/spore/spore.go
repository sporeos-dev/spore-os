package spore

import (
	"fmt"
	"log/slog"
	"spored/internal/manifest"
	"spored/internal/message"
	"spored/internal/pal"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"spored/internal/utilities/status"
	"strings"
)

type Spore struct {
	manifest *manifest.Manifest

	bus ibus
	hyphae ihyphae
	permissions ipermissions
	nodes inodes
}
	
func New() *Spore {
	m := &manifest.Manifest{
		Status: status.New(),
		Path: pal.FileSporeManifest(),
		ExpectedChecksum: "n/a",
	}
	m.Load()
	m.Status.Set(status.Verified)
	slog.Debug("Spore manifest loaded.", "manifest", m.Path)

	return &Spore{
		manifest: m,
	}
}

func (s *Spore) Set(bus ibus, hyphae ihyphae, nodes inodes, permissions ipermissions) {
	s.bus = bus
	s.hyphae = hyphae
	s.nodes = nodes
	s.permissions = permissions

	s.bus.Register(s)
}

func (s *Spore) Close() {
	s.bus.Unregister(s)
}

func (s *Spore) ReceiveCapture(response message.Message, args... any) *error.Error { return nil }

func (s *Spore) respond(request message.Message, args ...out.IOut) {
	parts := []string{fmt.Sprintf("~%s:%s", request.Handle(), request.Command())}
	for _, arg := range args {
		parts = append(parts, arg.String())
	}
	parts = append(parts, "ok")
	parts = append(parts, fmt.Sprintf("capture=%s", s.manifest.ID))
	wire := strings.Join(parts, " ")
	msg, err := message.Response(wire, s.manifest.ID)
	if err != nil {
		return
	}
	s.bus.Response(msg)
}

func (s *Spore) respondError(request message.Message, err *error.Error) {
	wire := fmt.Sprintf("~%s:%s %s capture=%s",
		request.Handle(),
		request.Command(),
		err.Wire(),
		s.manifest.ID)
	msg, parseErr := message.Response(wire, s.manifest.ID)
	if parseErr != nil {
		return
	}
	s.bus.Response(msg)
}

//
//
// INode
//

func (s *Spore) Id() string {
	return s.manifest.ID
}

func (s *Spore) IsConnected() bool {
	return true
}

func (s *Spore) IsWitness() bool {
	return false
}

func (s *Spore) GetManifest() *manifest.Manifest {
	return s.manifest
}

func (s *Spore) Receive(request message.Message) *error.Error {
	switch request.Command() {
	case "SPORE.help": return s.help(request)
	case "SPORE.info": return s.info(request)
	case "SPORE.state": return s.state(request)
		
	case "SPORE.node.list": return s.nodeList(request)
	case "SPORE.node.help": return s.nodeHelp(request)
	case "SPORE.node.state": return s.nodeState(request)
	case "SPORE.node.install": return s.nodeInstall(request)
	case "SPORE.node.uninstall": return s.nodeUninstall(request)
	case "SPORE.node.spawn": return s.nodeSpawn(request)
	case "SPORE.node.kill": return s.nodeKill(request)

	case "SPORE.command.list": return s.commandList(request)
	case "SPORE.command.help": return s.commandHelp(request)

	case "SPORE.error.list": return s.errorList(request)
	case "SPORE.error.help": return s.errorHelp(request)

	case "SPORE.topic.list": return s.topicList(request)
	case "SPORE.topic.help": return s.topicHelp(request)
	case "SPORE.topic.state": return s.topicState(request)
	case "SPORE.topic.subscribe": return s.topicSubscribe(request)
	case "SPORE.topic.unsubscribe": return s.topicUnsubscribe(request)

	case "SPORE.permissions.list": return s.permissionList(request)
	case "SPORE.permission.request": return s.permissionRequest(request)
	case "SPORE.permission.grant": return s.permissionGrant(request)
	case "SPORE.permission.revoke": return s.permissionRevoke(request)

	case "SPORE.security.keyring.list": return s.securityKeyringList(request)
	case "SPORE.security.keyring.info": return s.securityKeyringInfo(request)
	case "SPORE.security.keyring.grant": return s.securityKeyringGrant(request)
	case "SPORE.security.keyring.revoke": return s.securityKeyringRevoke(request)
	case "SPORE.security.signature.sign": return s.securitySignatureSign(request)
	case "SPORE.security.signature.verify": return s.securitySignatureVerify(request)
	}

	return error.New(
		error.Missing,
		error.Spore,
		"spore command unhandled",
		out.Pair("command", request.Command()))
}