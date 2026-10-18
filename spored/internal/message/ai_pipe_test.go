// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package message

import (
	"strings"
	"testing"
)

func TestSplitPipeIgnoresPipesInBlockValues(t *testing.T) {
	raw := "~s3c42:alias.run error code=Execution what=\"failed to run alias\" output=<<\n" +
		"def project_root() -> Path | None:\n" +
		">> node_error"

	if IsPipe(raw) {
		t.Fatalf("IsPipe(%q) = true, want false", raw)
	}

	parts := splitPipe(raw)
	if len(parts) != 1 {
		t.Fatalf("splitPipe() returned %d parts, want 1: %q", len(parts), parts)
	}
	if !strings.Contains(parts[0], "Path | None") {
		t.Fatalf("splitPipe() lost block content: %q", parts[0])
	}
}

func TestSplitPipeRecognizesPipeAfterBlockValue(t *testing.T) {
	raw := "source output=<<\nPath | None\n>> | destination"

	parts := splitPipe(raw)
	if len(parts) != 2 {
		t.Fatalf("splitPipe() returned %d parts, want 2: %q", len(parts), parts)
	}
	if parts[0] != "source output=<<\nPath | None\n>>" {
		t.Errorf("splitPipe()[0] = %q, want block message", parts[0])
	}
	if parts[1] != "destination" {
		t.Errorf("splitPipe()[1] = %q, want %q", parts[1], "destination")
	}
}
