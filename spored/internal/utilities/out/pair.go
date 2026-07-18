package out

import "fmt"

type Pair struct {
	Key string
	Value string
}

func NewPair(key string, value string) *Pair {
	return &Pair {
		Key: key,
		Value: value,
	}
}

func (p *Pair) String() string {
	return fmt.Sprintf(`%s="%s"`, p.Key, p.Value)
}
