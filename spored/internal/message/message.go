package message

import "spored/internal/utilities/error"

type Message interface{
	IsWitness() bool
	Get() string
	ToJSON() string

	Command() string
	Cast() string
	Arg(key string) (string, *error.Error)
	ArgIf(key string, ifnot string) string
	Flag(flag string) bool
	Handle() string

	Topic() string
}
