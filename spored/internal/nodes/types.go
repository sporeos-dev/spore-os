package nodes

import (
	"bufio"
	"net"
	"spored/internal/utilities/error"
)



type trust string
const (
	System trust = "system"
	Developer trust = "developer"
	Trusted trust = "trusted"
	StandardTrust trust = "standard"
	Untrusted trust = "untrusted"
)

type risk string
const (
	Benign risk = "benign"
	StandardRisk risk = "standard"
	Personal risk = "personal"
	Secret risk = "secret"
	Protected risk = "protected"
)

type ibus interface {}

type ihyphae interface {}

type ispore interface {}

type inode interface {
	id() string
	handleConnection(net.Conn, *bufio.Reader, *bufio.Writer) *error.Error
}
