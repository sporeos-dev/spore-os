package message

import (
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
)

type broadcast struct {
	raw string
	id string

	topic string 
	args map[string]string
	flags []string
}

func Broadcast(raw string, id string) (Message, *error.Error) {
	parsed, err := parseBroadcast(raw)
	if err != nil {
		return nil, err
	}
	b := &broadcast{
		raw: raw,
		id: id,
		topic: parsed.command,
		args: parsed.args,
		flags: parsed.flags,
	}
	return b, nil
}

func (b *broadcast) Get() string {
	return b.raw
}

func (b *broadcast) IsWitness() bool {
	return false
}

// Command returns the topic subject (the token after "publish").
func (b *broadcast) Command() string {
	return b.topic
}

// Cast returns the publishing node's ID.
func (b *broadcast) Cast() string {
	return b.id
}

func (b *broadcast) Arg(key string) (string, *error.Error) {
	if value, ok := b.args[key]; ok {
		return value, nil
	}
	return "", error.New(error.Missing, error.Message, "key not found", out.Pair("key", key))
}

func (b *broadcast) ArgIf(key string, ifnot string) string {
	if value, ok := b.args[key]; ok {
		return value
	}
	return ifnot
}

func (b *broadcast) Flag(flag string) bool {
	for _, f := range b.flags {
		if f == flag {
			return true
		}
	}
	return false
}

func (b *broadcast) Handle() string {
	return "n/a"
}

func (b *broadcast) Topic() string {
	return b.topic
}
