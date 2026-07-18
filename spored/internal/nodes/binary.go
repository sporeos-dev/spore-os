package nodes

import (
	"net"
	"spored/internal/pal"
	"spored/internal/registry"
	"spored/internal/utilities/file"
	"spored/internal/utilities/status"
)

type binary struct {
	status status.Status
	path string
	expectedChecksum string
}

func newBinary(registry *registry.Element) *binary {
	b := &binary{
		status: status.New(),
		path: registry.Binary,
		expectedChecksum: registry.BinaryChecksum,
	}

	if file.Exists(b.path) == false {
		b.status.Set(status.Missing)
	}

	return b
}

func (b *binary) close() {}

func (b *binary) verify(conn net.Conn) {
	if file.IsReadable(b.path) == false {
		b.status.Set(status.RequiresUserSpace)
		return
	}

	processPath, err := pal.ProcessPath(conn)
	if err != nil {
		b.status.Set(status.FailedChecksum)
		return
	}

	if b.path != processPath {
		b.status.Set(status.FailedChecksum)
		return
	}

	checksum, err := file.CalculateChecksum(processPath)
	if err != nil {
		b.status.Set(status.FailedChecksum)
		return
	}
	if checksum != b.expectedChecksum {
		b.status.Set(status.FailedChecksum)
		return
	}

	b.status.Set(status.Verified)
}