package message

import (
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
)

type response struct {
	raw string
	id string 

	command string
	args map[string]string
	flags []string
	handle string
}

func Response(raw string, captureID string) (Message, *error.Error) {
	parsed, err := parseResponse(raw)
	if err != nil {
		return nil, err
	}
	r := &response{
		raw:     raw,
		id:      captureID,
		command: parsed.command,
		args:    parsed.args,
		flags:   parsed.flags,
		handle:  parsed.handle,
	}
	return r, nil
}

func (r *response) Get() string {
	return r.raw
}

func (r *response) IsWitness() bool {
	return false
}

// Command returns the echoed subject from the ~handle:subject prefix.
func (r *response) Command() string {
	return r.command
}

// Cast returns the responding node's ID (the capture).
func (r *response) Cast() string {
	return r.id
}

func (r *response) Arg(key string) (string, *error.Error) {
	if value, ok := r.args[key]; ok {
		return value, nil
	}
	return "", error.New(error.Missing, error.Message, "key not found", out.Pair("key", key))
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

func (r *response) Topic() string {
	return "n/a"
}
