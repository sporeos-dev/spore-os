// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package message

import (
	"spored/internal/utilities/out"
	"testing"
)

func TestSporeResponsePreservesMultilinePersistedState(t *testing.T) {
	request, ok := Request(
		"dev.sporeos.SPORE.state.retrieve key=state ~alias-load",
		"dev.sporeos.foundry.alias",
	)
	if !ok {
		t.Fatal("Request returned false")
	}

	body := `scripts:
    hipy:
        alias: hipy
        path: /Users/mh/Documents/dev/foundry-alias/test/hello.py
        description: Testing Python.`
	response := Spore(request, out.Pair("body", body))

	parsed, ok := Response(response.Wire(), request.Cast())
	if !ok {
		t.Fatalf("Response could not parse multiline state:\n%s", response.Wire())
	}

	actual, ok := parsed.Arg("body")
	if !ok {
		t.Fatal("response does not contain body")
	}
	if actual != body {
		t.Errorf("body = %q, want %q", actual, body)
	}
}
