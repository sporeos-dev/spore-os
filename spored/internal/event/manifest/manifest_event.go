package manifest_event

import "spored/internal/manifest"

// manifest announcement
type IManifestAnnouncementContext interface {
	Manifest() *manifest.Manifest
}

type IManifestAnnouncmentHandler interface {
	OnManifestAnnouncement(ctx IManifestAnnouncementContext)
}

// manifest withdrawal
type IManifestWithdrawalContext interface {
	Manifest() *manifest.Manifest
}

type IManifestWithdrawalHandler interface {
	OnManifestWithdrawal(ctx IManifestWithdrawalContext)
}

// ----- implementation

type manifestWrap struct {
	m *manifest.Manifest
}

func New(m *manifest.Manifest) *manifestWrap {
	return &manifestWrap{m: m}
}

func (m *manifestWrap) Manifest() *manifest.Manifest {
	return m.m
}


