package spore

type Spore struct {}

func New() *Spore {
	return &Spore{}
}

func (s *Spore) Close() {}
