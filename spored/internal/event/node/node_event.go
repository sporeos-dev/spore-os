package node_event

import "spored/internal/manifest"

type INodeInstallationContext interface {
	Manifest() *manifest.Manifest	
}

type INodeInstallationHandler interface {
	OnNodeInstallation(ctx INodeInstallationContext)
}

type INodeUninstallationHandler interface {
	OnNodeUninstallation(id string)
}

type INodeConnectionHandler interface {
	OnNodeConnection(id string)
}

type INodeDisconnectionHandler interface {
	OnNodeDisconnection(id string)
}

type manifestWrap struct {
	m *manifest.Manifest
}

func New(m *manifest.Manifest) *manifestWrap {
	return &manifestWrap{
		m: m,
	}
}

func (m *manifestWrap) Manifest() *manifest.Manifest {
	return m.m
}


