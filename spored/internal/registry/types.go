package registry

type imanifest interface {
	GetId() string
	GetName() string
	GetTrust() string
	GetManifestPath() string
	GetManifestChecksum() string
	GetBinaryPath() string
}
