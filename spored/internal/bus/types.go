package bus

type WitnessType int
const (
	Incoming WitnessType = iota
	Outgoing
	Spore
	Node
)

type ihyphae interface {}

type inodes interface {}

type inode interface {
	Id() string
	IsWitness() bool
	SendRaw(msg string)
}

type ispore interface {}
