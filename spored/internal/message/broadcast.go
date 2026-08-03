package message

import (
	"fmt"
	"time"
)

type broadcast struct {
	raw string
	id string

	topic string 
	args map[string]string
	flags []string

	incomingSent bool
}

func Broadcast(raw string, id string) (Message, bool) {
	parsed, ok := parseBroadcast(raw)
	if !ok {
		return nil, false
	}
	b := &broadcast{
		raw: raw,
		id: id,
		topic: parsed.command,
		args: parsed.args,
		flags: parsed.flags,
	}
	return b, true
}

func (b *broadcast) Capability() string {
	return b.topic
}

// Cast returns the publishing node's ID.
func (b *broadcast) Cast() string {
	return b.id
}

func (b *broadcast) Capture() string {
	return "n/a"
}

func (b *broadcast) Arg(key string) (string, bool) {
	if value, ok := b.args[key]; ok {
		return value, true
	}
	return "", false
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

// func (b *broadcast) ToJSON() string {
// 	result := map[string]interface{}{
// 		"topic": b.topic,
// 		"cast":  b.id,
// 		"args":  b.args,
// 		"flags": b.flags,
// 	}
// 	data, err := json.Marshal(result)
// 	if err != nil {
// 		return "{}"
// 	}
// 	return string(data)
// }

// raw
// --> publish topic arg=value flag
// wire: +cast
// --> <raw> +cast
func (b *broadcast) Wire() string {
	return fmt.Sprintf(`%s cast=%s`, b.raw, b.id)
}

// witness in: +witness +spore_incoming +spore_time
// --> witness <wire> spore_incoming spore_time=time
// witness out: +witness +spore_outgoing +spore_time
// --> witness <wire> spore_outgoing spore_time=time
func (b *broadcast) Witness() string {
	t := time.Now().UnixMilli()
	incomingSent := b.incomingSent
	b.incomingSent = true
	witnessFlag := "spore_incoming"
	if incomingSent {
		witnessFlag = "spore_outgoing"
	}
	return fmt.Sprintf(`witness %s %s spore_time=%d`, b.Wire(), witnessFlag, t)
}
