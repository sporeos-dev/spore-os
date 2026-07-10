package policy

type Policy struct {}

func New() *Policy {
	return &Policy{}
}

func (p *Policy) Close() {}
