package node_event

import "spored/internal/manifest"

type INodeInstallationContext interface {
	Manifest() *manifest.Manifest
	SetInstalled()
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
	m         *manifest.Manifest
	installed bool
}

func New(m *manifest.Manifest) *manifestWrap {
	return &manifestWrap{
		m: m,
	}
}

func (m *manifestWrap) Manifest() *manifest.Manifest {
	return m.m
}

func (m *manifestWrap) SetInstalled() {
	m.installed = true
}

func (m *manifestWrap) Installed() bool {
	return m.installed
}
