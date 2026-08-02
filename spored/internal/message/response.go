package message

import (
	"fmt"
	"time"
)

type response struct {
	raw string
	id string 

	command string
	args map[string]string
	flags []string
	handle string
}

func Response(raw string, captureID string) (Message, bool) {
	parsed, ok := parseResponse(raw)
	if !ok {
		return nil, false
	}
	r := &response{
		raw:     raw,
		id:      captureID,
		command: parsed.command,
		args:    parsed.args,
		flags:   parsed.flags,
		handle:  parsed.handle,
	}
	return r, true
}

func (r *response) Capability() string {
	return r.command
}

// Cast returns the responding node's ID (the capture).
func (r *response) Cast() string {
	return r.ArgIf("cast", "n/a")
}

func (r *response) Capture() string {
	return r.id
}

func (r *response) Arg(key string) (string, bool) {
	if value, ok := r.args[key]; ok {
		return value, true
	}
	return "", false
}

func (r *response) ArgIf(key string, ifnot string) string {
	if value, ok := r.args[key]; ok {
		return value
	}
	return ifnot
}

func (r *response) Flag(flag string) bool {
	for _, f := range r.flags {
		if f == flag {
			return true
		}
	}
	return false
}

func (r *response) Handle() string {
	return r.handle
}

// func (r *response) ToJSON() string {
// 	result := map[string]interface{}{
// 		"command": r.command,
// 		"handle":  r.handle,
// 		"capture": r.id,
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
// --> ~handle:subject arg=val flag cast=requester.id
// wire: +ok +capture
// --> <raw> ok capture=responder.id
func (r *response) Wire() string {
	return fmt.Sprintf(`%s ok cature=%s`, r.raw, r.id)
}

// witness out: +witness +spore_outgoing +spore_time
// --> witness <wire> spore_outgoing spore_time=time
func (r *response) Witness() string {
	t := time.Now().UnixMilli()
	return fmt.Sprintf(`witness %s spore_outgoing spore_time=%d`, r.Wire(), t)
}
