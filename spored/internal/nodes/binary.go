package nodes

import (
	"net"
	"spored/internal/pal"
	"spored/internal/utilities/file"
)

type binary struct {
	status status
	path string
	expectedChecksum string
}

func newBinary(registry *registryElement) *binary {
	b := &binary{
		status: newStatus(),
		path: registry.Binary,
		expectedChecksum: registry.BinaryChecksum,
	}

	if file.Exists(b.path) == false {
		b.status.set(Missing)
	}

	return b
}

func (b *binary) close() {}

func (b *binary) verify(conn net.Conn) {
	if file.IsReadable(b.path) == false {
		b.status.set(RequiresUserSpace)
		return
	}

	processPath, err := pal.ProcessPath(conn)
	if err != nil {
		b.status.set(FailedChecksum)
		return
	}

	if b.path != processPath {
		b.status.set(FailedChecksum)
		return
	}

	checksum, err := file.CalculateChecksum(processPath)
	if err != nil {
		b.status.set(FailedChecksum)
		return
	}
	if checksum != b.expectedChecksum {
		b.status.set(FailedChecksum)
		return
	}

	b.status.set(Verified)
}