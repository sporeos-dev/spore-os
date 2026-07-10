package spore

import (
	"encoding/json"
	"errors"
	"spored/internal/message"
)

func (s *Spore) errorList(msg *message.Spore) (string, error) {
	nodeid := msg.GetArgumentIf("node", "n/a")

	var errCodes []string

	switch nodeid {
	case "dev.sporeos.SPORE":
		// Hub node — standard errors only.
		for _, info := range message.StandardErrors() {
			errCodes = append(errCodes, string(info.Code))
		}
	case "n/a":
		// No node — standard errors plus every registered node's custom errors.
		for _, info := range message.StandardErrors() {
			errCodes = append(errCodes, string(info.Code))
		}
		for _, nid := range s.hub.ListNodes() {
			node, err := s.hub.GetNode(nid)
			if err != nil {
				continue
			}
			for _, e := range node.GetManifest().Errors {
				errCodes = append(errCodes, e.Name)
			}
		}
	default:
		// Specific node — custom errors for that node only.
		node, err := s.hub.GetNode(nodeid)
		if err != nil {
			return "", err
		}
		for _, e := range node.GetManifest().Errors {
			errCodes = append(errCodes, e.Name)
		}
	}

	println()
	println("Errors:")
	for _, e := range errCodes {
		println(" -", e)
	}
	println()

	serialized, err := json.Marshal(errCodes)
	if err != nil {
		return "", err
	}

	return s.returnCapture(msg, map[string]string{
		"errors": string(serialized),
	}, []string{}), nil
}

func (s *Spore) errorHelp(msg *message.Spore) (string, error) {
	name, err := msg.GetArgument("error")
	if err != nil {
		return "", err
	}

	// Check standard errors first.
	for _, info := range message.StandardErrors() {
		if string(info.Code) == name {
			println()
			println(" "+string(info.Code))
			println(" "+info.Description)
			if len(info.Examples) > 0 {
				println(" Examples:")
				for _, ex := range info.Examples {
					println("   -", ex)
				}
			}
			println()

			result := map[string]interface{}{
				"name":        string(info.Code),
				"description": info.Description,
				"examples":    info.Examples,
			}
			serialized, err := json.Marshal(result)
			if err != nil {
				return "", err
			}
			return s.returnCapture(msg, map[string]string{
				"errorinfo": string(serialized),
			}, []string{}), nil
		}
	}

	// Not a standard code — search manifest-declared errors in registered nodes.
	for _, nodeid := range s.hub.ListNodes() {
		node, err := s.hub.GetNode(nodeid)
		if err != nil {
			continue
		}
		for _, e := range node.GetManifest().Errors {
			if e.Name == name {
				println()
				println(" "+e.Name)
				println(" "+e.Description)
				if len(e.Examples) > 0 {
					println(" Examples:")
					for _, ex := range e.Examples {
						println("   -", ex)
					}
				}
				println()

				result := map[string]interface{}{
					"name":        e.Name,
					"description": e.Description,
					"examples":    e.Examples,
				}
				serialized, err := json.Marshal(result)
				if err != nil {
					return "", err
				}
				return s.returnCapture(msg, map[string]string{
					"errorinfo": string(serialized),
				}, []string{}), nil
			}
		}
	}

	return "", errors.New("error code not found: " + name)
}