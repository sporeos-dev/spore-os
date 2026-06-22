// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package manifest

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Manifest struct {
	ID          string          `yaml:"id"`
	Name        string          `yaml:"name"`
	Description string          `yaml:"description"`
	Schema      string          `yaml:"schema"`
	Version     string          `yaml:"version"`
	App         string          `yaml:"app"`
	Autostart   bool            `yaml:"autostart"`
	Witness		bool			`yaml:"witness"`
	Api         []Command       `yaml:"api"`
	Errors      []ManifestError `yaml:"errors"`

	Path 		string `yaml:"-"`
}

type Command struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	Usage       []string  `yaml:"usage"`
	Inputs      *[]Input  `yaml:"inputs"`
	Outputs     *[]Output `yaml:"outputs"`
	Notes       *[]string `yaml:"notes"`
}

type Input struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
}

type Output struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
}

type ManifestError struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Examples    []string `yaml:"examples,omitempty"`
}

// reservedInputNames are keywords that may not be used as input argument names.
// These are hub-injected routing fields and caller-behavior flags whose presence
// on an inbound call has protocol meaning.
// Note: all names with the spore_ prefix are also reserved (checked separately).
var reservedInputNames = map[string]bool{
	"cast": true, "capture": true,
	"code": true, "what": true, "ok": true, "json": true,
}

// reservedOutputNames are keywords that may not be used as output field names.
// Includes all input-reserved names plus the response status flags (error,
// custom_error, cancelled) which would be indistinguishable from protocol
// response markers if a node emitted them as data fields.
// Note: all names with the spore_ prefix are also reserved (checked separately).
var reservedOutputNames = map[string]bool{
	"cast": true, "capture": true,
	"code": true, "what": true, "ok": true, "json": true,
	"error": true, "custom_error": true, "cancelled": true,
}

func validateManifest(m *Manifest) error {
	// SPEC §2.4: No third-party node may register an id beginning with SPORE.
	if strings.HasPrefix(m.ID, "SPORE.") {
		return fmt.Errorf("manifest id %q uses the reserved SPORE. namespace", m.ID)
	}

	for _, cmd := range m.Api {
		// SPEC §6.6: 'witness' is a reserved line-prefix token on the wire.
		// No subject may be named 'witness'.
		if cmd.Name == "witness" || strings.HasPrefix(cmd.Name, "witness.") {
			return fmt.Errorf("command %q uses the reserved 'witness' name", cmd.Name)
		}

		if cmd.Inputs != nil {
			for _, input := range *cmd.Inputs {
				// SPEC §6.6: all spore_-prefixed names are reserved.
				if reservedInputNames[input.Name] || strings.HasPrefix(input.Name, "spore_") {
					return fmt.Errorf("command %s: input %q is a reserved argument name", cmd.Name, input.Name)
				}
			}
		}
		if cmd.Outputs != nil {
			for _, output := range *cmd.Outputs {
				// SPEC §6.6: all spore_-prefixed names are reserved.
				if reservedOutputNames[output.Name] || strings.HasPrefix(output.Name, "spore_") {
					return fmt.Errorf("command %s: output %q is a reserved argument name", cmd.Name, output.Name)
				}
			}
		}
	}
	return nil
}

func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var manifest Manifest
	manifest.Path = path
	err = yaml.Unmarshal(data, &manifest)
	if err != nil {
		return nil, err
	}

	if err := validateManifest(&manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}
