package registry

type imanifest interface {
	GetId() string
	GetName() string
	GetManifestPath() string
	GetManifestChecksum() string
	GetBinaryPath() string
}
