package message

import (
	"encoding/json"
	"fmt"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
)

type request struct {
	raw string
	id  string
	
	command string
	args map[string]string
	flags []string
	handle string
}

func Request(raw string, id string) (Message, *error.Error) {
	parsed, err := parseRequest(raw)
	if err != nil {
		return nil, err
	}
	r := &request{
		raw: raw,
		id: id,
		command: parsed.command,
		args: parsed.args,
		flags: parsed.flags,
		handle: parsed.handle,
	}

	return r, nil
}

func (r *request) Get() string {
	return fmt.Sprintf(`%s cast=%s`, r.raw, r.id)
}

func (r *request) IsWitness() bool {
	return false
}

func (r *request) Command() string {
	return r.command
}

func (r *request) Cast() string {
	return r.id
}

func (r *request) Arg(key string) (string, *error.Error) {
	if value, ok := r.args[key]; ok {
		return value, nil
	}
	return "", error.New(error.Missing, error.Message, "key not found", out.Pair("key", key))
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

func (r *request) ToJSON() string {
	result := map[string]interface{}{
		"command": r.command,
		"handle":  r.handle,
		"cast":    r.id,
		"args":    r.args,
		"flags":   r.flags,
	}
	data, err := json.Marshal(result)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func (r *request) Topic() string {
	return "n/a"
}