package nodes

import (
	"bufio"
	"log/slog"
	"net"
	"spored/internal/utilities/error"
)

type spore struct {
	spore ispore
}

func newSpore(s ispore) inode {
	slog.Debug("Creating central spore node", "name", "Spore OS")

	spore := &spore {
		spore: s,
	}

	return spore
}

//
//
// inode
//

func (s *spore) id() string {
	return "dev.sporeos.SPORE"
}

func (s *spore) handleConnection(conn net.Conn, reader *bufio.Reader, writer *bufio.Writer) *error.Error {
	return error.New(error.HandshakeDenial, "cannot handshake spore")
}
