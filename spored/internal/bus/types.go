package bus

type WitnessType int
const (
	Incoming WitnessType = iota
	Outgoing
	Spore
	Node
)

type node interface {
	Id() string
	IsWitness() bool
	SendRaw(msg string)
}

type message interface {}