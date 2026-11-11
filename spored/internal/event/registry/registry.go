package registry_event

import "spored/internal/registry"

type IRegistryContext interface {
	Registry() *registry.Element
}

type IRegistryAnnouncementHandler interface {
	OnRegistryAnnouncement(ctx IRegistryContext)
}

type IRegistryWithdrawalHandler interface {
	OnRegistryWithdrawal(ctx IRegistryContext)
}

type reg struct {
	elem *registry.Element
}

func New(elem *registry.Element) *reg {
	return &reg{
		elem: elem,
	}
}

func (r *reg) Registry() *registry.Element {
	return r.elem
}

