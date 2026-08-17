package message

import (
	"fmt"
	"spored/internal/iface"
	"time"
)

type request struct {
	raw string
	id  string
	
	command string
	args map[string]string
	flags []string
	handle string
}

func Request(raw string, id string) (iface.Message, bool) {
	parsed, ok := parseRequest(raw)
	if !ok {
		return nil, false
	}
	r := &request{
		raw: raw,
		id: id,
		command: parsed.command,
		args: parsed.args,
		flags: parsed.flags,
		handle: parsed.handle,
	}

	return r, true
}

func (r *request) Capability() string {
	return r.command
}

func (r *request) Cast() string {
	return r.id
}

func (r *request) Capture() string {
	return "n/a"
}

func (r *request) Arg(key string) (string, bool) {
	if value, ok := r.args[key]; ok {
		return value, true
	}
	return "", false
}

func (r *request) ArgIf(key string, ifnot string) string {
	if value, ok := r.args[key]; ok {
		return value
	}
	return ifnot
}

func (r *request) Flag(flag string) bool {
	for _, f := range r.flags {
		if f == flag {
			return true
		}
	}
	return false
}

func (r *request) Handle() string {
	return r.handle
}

// func (r *request) ToJSON() string {
// 	result := map[string]interface{}{
// 		"command": r.command,
// 		"handle":  r.handle,
// 		"cast":    r.id,
// 		"args":    r.args,
// 		"flags":   r.flags,
// 	}
// 	data, err := json.Marshal(result)
// 	if err != nil {
// 		return "{}"
// 	}
// 	return string(data)
// }

// raw
// --> subject arg=val flag ~handle
// wire: +cast
// --> <raw> cast=requester.id
func (r *request) Wire() string {
	return fmt.Sprintf("%s cast=%s", r.raw, r.id)
}

// witness in: +witness +spore_incoming +spore_time 
// --> witness <wire> spore_incoming spore_time=time
func (r *request) Witness() string {
	t := time.Now().UnixMilli()
	return fmt.Sprintf("witness body='%s' spore_incoming spore_time=%d", r.Wire(), t)
}
