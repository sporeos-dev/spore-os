package message

type Type int
const (
	WitnessIncoming Type = iota
	WitnessOutgoing
	WitnessSpore
	WitnessNode
)

type inode interface {
	Id() string
}