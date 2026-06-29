package spore

import (
	"encoding/json"
	"errors"
	"spored/internal/message"
)

func (s *Spore) help(msg *message.Spore) (string, error) {

	if s.manifest == nil {
		return "", errors.New("SPORE manifest not loaded")
	}

	// from SPORE
	manifest := s.manifest

	// show node help
	println()
	println("  " + manifest.Name + " // " + manifest.ID)
	println("  " + manifest.Description)
	println("  Version: " + manifest.Version + " // Schema: " + manifest.Schema)
	println("  " + manifest.App)
	println("  ----------")
	
	println("  Api:")
	api := []string{}
	for _, command := range manifest.Api {
		println("  -", command.Name)
		api = append(api, command.Name)
	}
	println()

	// serialize and return
	result := map[string]interface{}{
		"id": manifest.ID,
		"name": manifest.Name,
		"description": manifest.Description,
		"version": manifest.Version,
		"schema": manifest.Schema,
		"api": api,
	}

	serialized, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return s.returnCapture(msg, map[string]string{
		"nodeinfo": string(serialized),
	}, []string{}), nil
}
