// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package spore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPersistFileWritesPrivateState(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "data")
	filename, persistError := persistFileName(directory, "preferences", "dev.sporeos.example")
	if persistError != nil {
		t.Fatalf("persistFileName returned an error: %v", persistError)
	}

	if persistError := persistFile(filename, []byte("saved state")); persistError != nil {
		t.Fatalf("persistFile returned an error: %v", persistError)
	}

	body, readError := os.ReadFile(filename)
	if readError != nil {
		t.Fatalf("ReadFile returned an error: %v", readError)
	}
	if string(body) != "saved state" {
		t.Errorf("persisted body = %q, want %q", body, "saved state")
	}

	directoryInfo, statError := os.Stat(directory)
	if statError != nil {
		t.Fatalf("Stat directory returned an error: %v", statError)
	}
	if directoryInfo.Mode().Perm() != 0700 {
		t.Errorf("directory mode = %o, want 0700", directoryInfo.Mode().Perm())
	}

	fileInfo, statError := os.Stat(filename)
	if statError != nil {
		t.Fatalf("Stat persisted file returned an error: %v", statError)
	}
	if fileInfo.Mode().Perm() != 0600 {
		t.Errorf("file mode = %o, want 0600", fileInfo.Mode().Perm())
	}
}

func TestPersistFileResecuresExistingFile(t *testing.T) {
	directory := t.TempDir()
	filename := filepath.Join(directory, "existing.persist")
	if writeError := os.WriteFile(filename, []byte("old state"), 0644); writeError != nil {
		t.Fatalf("WriteFile returned an error: %v", writeError)
	}

	if persistError := persistFile(filename, []byte("new state")); persistError != nil {
		t.Fatalf("persistFile returned an error: %v", persistError)
	}

	fileInfo, statError := os.Stat(filename)
	if statError != nil {
		t.Fatalf("Stat returned an error: %v", statError)
	}
	if fileInfo.Mode().Perm() != 0600 {
		t.Errorf("file mode = %o, want 0600", fileInfo.Mode().Perm())
	}
}

func TestPersistFileNameRejectsPathSeparators(t *testing.T) {
	for _, testCase := range []struct {
		name string
		key  string
		cast string
	}{
		{name: "key forward slash", key: "../preferences", cast: "dev.sporeos.example"},
		{name: "key backslash", key: `..\preferences`, cast: "dev.sporeos.example"},
		{name: "cast forward slash", key: "preferences", cast: "../dev.sporeos.example"},
		{name: "empty key", key: "", cast: "dev.sporeos.example"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, persistError := persistFileName(t.TempDir(), testCase.key, testCase.cast)
			if persistError == nil {
				t.Fatal("persistFileName returned nil error")
			}
		})
	}
}
