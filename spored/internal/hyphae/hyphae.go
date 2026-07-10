package hyphae

type Hyphae struct {}

func New() *Hyphae {
	return &Hyphae {}
}

func (h *Hyphae) Close() {}
