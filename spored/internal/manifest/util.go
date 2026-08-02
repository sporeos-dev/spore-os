package manifest

import "strings"

// reservedArgNames are protocol keywords that may not be used as input/output names.
var reservedArgNames = map[string]bool{
	"cast": true, "capture": true, "ok": true, "error": true,
	"code": true, "what": true, "module": true,
	"spore_incoming": true, "spore_outgoing": true, "spore_event": true, "spore_node": true, "spore_time": true,
	"json": true,
}

// ValidateReservedLanguage returns an error string if the manifest violates any
// reserved-language rule: SPORE. ID prefix, reserved argument names, or reserved
// subject names ("witness", "publish").
func (m *Manifest) ValidateReservedLanguage() string {
	if strings.HasPrefix(m.ID, "SPORE.") {
		return "node id uses reserved SPORE. namespace: " + m.ID
	}
	for _, cmd := range m.Api {
		if cmd.Name == "witness" || strings.HasPrefix(cmd.Name, "witness ") {
			return "subject name is reserved: " + cmd.Name
		}
		if cmd.Name == "publish" || strings.HasPrefix(cmd.Name, "publish ") {
			return "subject name is reserved: " + cmd.Name
		}
		if err := checkArgs(cmd.Inputs); err != "" {
			return err
		}
		if err := checkOutputs(cmd.Outputs); err != "" {
			return err
		}
	}
	for _, topic := range m.Topics {
		if topic.Name == "witness" || strings.HasPrefix(topic.Name, "witness ") {
			return "topic name is reserved: " + topic.Name
		}
		if topic.Name == "publish" || strings.HasPrefix(topic.Name, "publish ") {
			return "topic name is reserved: " + topic.Name
		}
		if err := checkOutputs(topic.Outputs); err != "" {
			return err
		}
	}
	return ""
}

func checkArgs(inputs *[]Input) string {
	if inputs == nil {
		return ""
	}
	for _, in := range *inputs {
		if strings.HasPrefix(in.Name, "spore_") || reservedArgNames[in.Name] {
			return "reserved argument name in input: " + in.Name
		}
	}
	return ""
}

func checkOutputs(outputs *[]Output) string {
	if outputs == nil {
		return ""
	}
	for _, out := range *outputs {
		if strings.HasPrefix(out.Name, "spore_") || reservedArgNames[out.Name] {
			return "reserved argument name in output: " + out.Name
		}
	}
	return ""
}

func (m *Manifest) CommandIds() []string {
	ids := make([]string, 0, len(m.Api))
	for _, el := range m.Api {
		ids = append(ids, el.Name)
	}
	return ids
}

func (m *Manifest) TopicIds() []string {
	ids := make([]string, 0, len(m.Topics))
	for _, el := range m.Topics {
		ids = append(ids, el.Name)
	}
	return ids
}

func (m *Manifest) ErrorIds() []string {
	ids := make([]string, 0, len(m.Errors))
	for _, el := range m.Errors {
		ids = append(ids, el.Name)
	}
	return ids
}

