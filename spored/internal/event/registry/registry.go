package registry_event

import (
	reg_el "spored/internal/registry/element"
)

type IRegistryContext interface {
	Registry() *reg_el.Element
}

type IRegistryAnnouncementHandler interface {
	OnRegistryAnnouncement(ctx IRegistryContext)
}

type IRegistryWithdrawalHandler interface {
	OnRegistryWithdrawal(ctx IRegistryContext)
}

type reg struct {
	elem *reg_el.Element
}

func New(elem *reg_el.Element) *reg {
	return &reg{
		elem: elem,
	}
}

func (r *reg) Registry() *reg_el.Element {
	return r.elem
}

