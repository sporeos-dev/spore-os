// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

// =============================================================================
// SPEC §6.6: Reserved Keywords
// "The following argument names are reserved by the protocol. They may not be
// used as input or output names in any node manifest. The hub rejects manifests
// containing reserved keywords as argument names."
// Response status flags (error, custom_error, cancelled) are reserved as output
// names only — they may be used as input names (e.g. SPORE.error.help error=code).
// =============================================================================

func TestManifest_RejectsReservedInputNames(t *testing.T) {
	reserved := []string{
		"cast", "capture",
		"code", "what", "ok", "json",
		// SPEC §6.6: all spore_-prefixed names are reserved
		"spore_error", "spore_incoming", "spore_time", "spore_anything",
	}

	for _, name := range reserved {
		yaml := `
id: com.test.node
name: Test Node
description: A test node.
schema: SPORE/v1d0
app: ./test
api:
  - name: test.command
    description: A test command.
    usage:
      - test.command
    inputs:
      - name: ` + name + `
        type: string
        description: Should be rejected.
`
		path := writeTempManifest(t, yaml)
		_, err := LoadManifest(path)
		if err == nil {
			t.Errorf("expected error for reserved input name %q, but got nil", name)
		}
	}
}

func TestManifest_AllowsResponseStatusFlagsAsInputNames(t *testing.T) {
	// error, custom_error, cancelled are response status flags — reserved as outputs
	// but valid as input names (e.g. SPORE.error.help error=RouteNotFound)
	allowed := []string{"error", "custom_error", "cancelled"}

	for _, name := range allowed {
		yaml := `
id: com.test.node
name: Test Node
description: A test node.
schema: SPORE/v1d0
app: ./test
api:
  - name: test.command
    description: A test command.
    usage:
      - test.command
    inputs:
      - name: ` + name + `
        type: string
        description: Should be allowed.
`
		path := writeTempManifest(t, yaml)
		_, err := LoadManifest(path)
		if err != nil {
			t.Errorf("expected no error for input name %q, but got: %v", name, err)
		}
	}
}

func TestManifest_RejectsReservedOutputNames(t *testing.T) {
	reserved := []string{
		"cast", "capture", "error", "custom_error", "cancelled",
		"code", "what", "ok", "json",
		// SPEC §6.6: all spore_-prefixed names are reserved
		"spore_error", "spore_outgoing", "spore_time", "spore_anything",
	}

	for _, name := range reserved {
		yaml := `
id: com.test.node
name: Test Node
description: A test node.
schema: SPORE/v1d0
app: ./test
api:
  - name: test.command
    description: A test command.
    usage:
      - test.command
    outputs:
      - name: ` + name + `
        type: string
        description: Should be rejected.
`
		path := writeTempManifest(t, yaml)
		_, err := LoadManifest(path)
		if err == nil {
			t.Errorf("expected error for reserved output name %q, but got nil", name)
		}
	}
}

func TestManifest_AcceptsValidInputNames(t *testing.T) {
	yaml := `
id: com.test.clock
name: Clock
description: Returns time.
schema: SPORE/v1d0
app: ./clock
api:
  - name: get_time
    description: Returns the current time.
    usage:
      - clock.get_time
    inputs:
      - name: timezone
        type: string
        description: IANA timezone.
    outputs:
      - name: time
        type: string
        description: Current time.
`
	path := writeTempManifest(t, yaml)
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("valid manifest rejected: %v", err)
	}
	if m.ID != "com.test.clock" {
		t.Errorf("expected ID com.test.clock, got %q", m.ID)
	}
}

// =============================================================================
// SPEC §8.1: Manifest Structure
// =============================================================================

func TestManifest_LoadsAllMetadataFields(t *testing.T) {
	// SPEC §8.2: Required fields: id, name, app, schema, description
	yaml := `
id: com.example.clock
name: Clock
description: Provides time-related functions.
schema: SPORE/v1d0
version: 1.0.0
app: ./clock
api: []
`
	path := writeTempManifest(t, yaml)
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.ID != "com.example.clock" {
		t.Errorf("ID: got %q", m.ID)
	}
	if m.Name != "Clock" {
		t.Errorf("Name: got %q", m.Name)
	}
	if m.Description != "Provides time-related functions." {
		t.Errorf("Description: got %q", m.Description)
	}
	if m.Schema != "SPORE/v1d0" {
		t.Errorf("Schema: got %q", m.Schema)
	}
	if m.Version != "1.0.0" {
		t.Errorf("Version: got %q", m.Version)
	}
	if m.App != "./clock" {
		t.Errorf("App: got %q", m.App)
	}
}

func TestManifest_LoadsAPIEntries(t *testing.T) {
	// SPEC §8.3: API entry fields: name, description, usage, inputs, outputs
	yaml := `
id: com.example.clock
name: Clock
description: Time functions.
schema: SPORE/v1d0
app: ./clock
api:
  - name: get_time
    description: Returns the current time.
    usage:
      - clock.get_time
      - clock.get_time timezone=America/New_York
    inputs:
      - name: timezone
        type: string
        description: IANA timezone.
    outputs:
      - name: time
        type: string
        description: Current time in ISO 8601.
`
	path := writeTempManifest(t, yaml)
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.Api) != 1 {
		t.Fatalf("expected 1 API entry, got %d", len(m.Api))
	}

	cmd := m.Api[0]
	if cmd.Name != "get_time" {
		t.Errorf("API name: got %q", cmd.Name)
	}
	if cmd.Description != "Returns the current time." {
		t.Errorf("API description: got %q", cmd.Description)
	}
	if len(cmd.Usage) != 2 {
		t.Errorf("expected 2 usage examples, got %d", len(cmd.Usage))
	}
	if cmd.Inputs == nil || len(*cmd.Inputs) != 1 {
		t.Fatal("expected 1 input")
	}
	input := (*cmd.Inputs)[0]
	if input.Name != "timezone" || input.Type != "string" {
		t.Errorf("input: got name=%q type=%q", input.Name, input.Type)
	}
	if cmd.Outputs == nil || len(*cmd.Outputs) != 1 {
		t.Fatal("expected 1 output")
	}
	output := (*cmd.Outputs)[0]
	if output.Name != "time" || output.Type != "string" {
		t.Errorf("output: got name=%q type=%q", output.Name, output.Type)
	}
}

func TestManifest_LoadsCustomErrors(t *testing.T) {
	// SPEC §8.4: Custom error entries
	yaml := `
id: com.example.clock
name: Clock
description: Time functions.
schema: SPORE/v1d0
app: ./clock
api: []
errors:
  - name: clock.err.invalid_timezone
    description: The timezone is not valid.
    examples:
      - "Misspelled timezone"
      - "Empty string"
`
	path := writeTempManifest(t, yaml)
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.Errors) != 1 {
		t.Fatalf("expected 1 custom error, got %d", len(m.Errors))
	}
	e := m.Errors[0]
	if e.Name != "clock.err.invalid_timezone" {
		t.Errorf("error name: got %q", e.Name)
	}
	if len(e.Examples) != 2 {
		t.Errorf("expected 2 examples, got %d", len(e.Examples))
	}
}

func TestManifest_APIWithNoInputsOrOutputs(t *testing.T) {
	// SPEC §8.3: inputs/outputs are optional
	yaml := `
id: com.example.logger
name: Logger
description: Logs entries.
schema: SPORE/v1d0
app: ./logger
api:
  - name: flush
    description: Flushes the log buffer.
    usage:
      - logger.flush
`
	path := writeTempManifest(t, yaml)
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmd := m.Api[0]
	if cmd.Inputs != nil {
		t.Error("expected nil inputs for command with no inputs declared")
	}
	if cmd.Outputs != nil {
		t.Error("expected nil outputs for command with no outputs declared")
	}
}

func TestManifest_RecordsFilePath(t *testing.T) {
	yaml := `
id: com.example.test
name: Test
description: Test node.
schema: SPORE/v1d0
app: ./test
api: []
`
	path := writeTempManifest(t, yaml)
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Path != path {
		t.Errorf("expected path %q, got %q", path, m.Path)
	}
}

func TestManifest_RejectsInvalidYAML(t *testing.T) {
	path := writeTempManifest(t, "not: [valid: yaml: {{")
	_, err := LoadManifest(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

// =============================================================================
// SPEC §6.6: Reserved 'witness' subject name
// "No subject may be named 'witness'. The hub rejects manifests declaring such a subject."
// =============================================================================

func TestManifest_RejectsWitnessSubjectName(t *testing.T) {
	cases := []string{"witness", "witness.something"}

	for _, name := range cases {
		yaml := `
id: com.test.node
name: Test Node
description: A test node.
schema: SPORE/v1d0
app: ./test
api:
  - name: ` + name + `
    description: Should be rejected.
    usage:
      - ` + name + `
`
		path := writeTempManifest(t, yaml)
		_, err := LoadManifest(path)
		if err == nil {
			t.Errorf("expected error for reserved subject name %q, but got nil", name)
		}
	}
}


func TestManifest_RejectsNonexistentFile(t *testing.T) {
	_, err := LoadManifest("/nonexistent/path/manifest.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

// =============================================================================
// SPEC §8.2: Manifest Metadata — autostart field
// "If true, the hub spawns this node automatically on startup."
// =============================================================================

func TestManifest_AutostartDefaultsFalse(t *testing.T) {
	yaml := `
id: com.example.clock
name: Clock
description: Provides time-related functions.
schema: SPORE/v1d0
app: ./clock
api: []
`
	path := writeTempManifest(t, yaml)
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Autostart {
		t.Error("expected Autostart to default to false when omitted")
	}
}

func TestManifest_AutostartTrue(t *testing.T) {
	yaml := `
id: com.example.clock
name: Clock
description: Provides time-related functions.
schema: SPORE/v1d0
app: ./clock
autostart: true
api: []
`
	path := writeTempManifest(t, yaml)
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !m.Autostart {
		t.Error("expected Autostart to be true")
	}
}

// --- Helper ---

func writeTempManifest(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.manifest.spore.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp manifest: %v", err)
	}
	return path
}

// =============================================================================
// SPEC §2.4: SPORE.* Namespace Rejection
// "No third-party node may register an id beginning with SPORE."
// =============================================================================

func TestManifest_RejectsSporeNamespaceID(t *testing.T) {
	ids := []string{"SPORE.evil", "SPORE.hub", "SPORE.anything.deep"}

	for _, id := range ids {
		yaml := `
id: ` + id + `
name: Bad Node
description: Tries to claim SPORE namespace.
schema: SPORE/v1d0
app: ./bad
api: []
`
		path := writeTempManifest(t, yaml)
		_, err := LoadManifest(path)
		if err == nil {
			t.Errorf("expected error for SPORE namespace id %q, but got nil", id)
		}
	}
}

func TestManifest_AllowsNonSporeIDs(t *testing.T) {
	ids := []string{"com.example.clock", "dev.sporeos.SPORE", "org.spore.tool", "SPORELIKE.node"}

	for _, id := range ids {
		yaml := `
id: ` + id + `
name: Good Node
description: Valid node.
schema: SPORE/v1d0
app: ./good
api: []
`
		path := writeTempManifest(t, yaml)
		_, err := LoadManifest(path)
		if err != nil {
			t.Errorf("expected no error for id %q, but got: %v", id, err)
		}
	}
}

// =============================================================================
// SPEC §8.3: Input required field
// "required: boolean — Whether this argument must be provided."
// =============================================================================

func TestManifest_ParsesRequiredField(t *testing.T) {
	yaml := `
id: com.example.clock
name: Clock
description: Time functions.
schema: SPORE/v1d0
app: ./clock
api:
  - name: get_time
    description: Returns the current time.
    usage:
      - clock.get_time
    inputs:
      - name: timezone
        type: string
        description: IANA timezone.
        required: true
      - name: format
        type: string
        description: Output format.
        required: false
      - name: locale
        type: string
        description: Locale override.
`
	path := writeTempManifest(t, yaml)
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inputs := *m.Api[0].Inputs
	if !inputs[0].Required {
		t.Error("expected timezone.Required to be true")
	}
	if inputs[1].Required {
		t.Error("expected format.Required to be false")
	}
	if inputs[2].Required {
		t.Error("expected locale.Required to default to false when omitted")
	}
}
