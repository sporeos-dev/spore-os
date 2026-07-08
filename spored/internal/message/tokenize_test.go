// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package message

import (
	"testing"
)

// =============================================================================
// Tokenize — happy paths
// =============================================================================

func TestTokenize_EmptyString(t *testing.T) {
	tokens, err := Tokenize("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 0 {
		t.Errorf("expected empty slice, got %v", tokens)
	}
}

func TestTokenize_SingleToken(t *testing.T) {
	tokens, err := Tokenize("clock.get_time")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 || tokens[0] != "clock.get_time" {
		t.Errorf("expected [clock.get_time], got %v", tokens)
	}
}

func TestTokenize_MultipleTokens(t *testing.T) {
	tokens, err := Tokenize("clock.get_time timezone=UTC ~h1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"clock.get_time", "timezone=UTC", "~h1"}
	if len(tokens) != len(want) {
		t.Fatalf("expected %v, got %v", want, tokens)
	}
	for i, w := range want {
		if tokens[i] != w {
			t.Errorf("[%d] expected %q, got %q", i, w, tokens[i])
		}
	}
}

func TestTokenize_MultipleSpacesBetweenTokens(t *testing.T) {
	tokens, err := Tokenize("a   b     c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 3 {
		t.Errorf("expected 3 tokens, got %v", tokens)
	}
}

func TestTokenize_LeadingAndTrailingWhitespace(t *testing.T) {
	tokens, err := Tokenize("  a b  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 2 || tokens[0] != "a" || tokens[1] != "b" {
		t.Errorf("expected [a b], got %v", tokens)
	}
}

func TestTokenize_TabsAsWhitespace(t *testing.T) {
	tokens, err := Tokenize("a\tb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 2 {
		t.Errorf("expected 2 tokens, got %v", tokens)
	}
}

func TestTokenize_DoubleQuotedStringWithSpaces(t *testing.T) {
	tokens, err := Tokenize(`what="hello world"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 || tokens[0] != `what="hello world"` {
		t.Errorf(`expected [what="hello world"], got %v`, tokens)
	}
}

func TestTokenize_SingleQuotedStringWithSpaces(t *testing.T) {
	tokens, err := Tokenize("what='hello world'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 || tokens[0] != "what='hello world'" {
		t.Errorf("expected [what='hello world'], got %v", tokens)
	}
}

func TestTokenize_DoubleQuoteContainingSingleQuote(t *testing.T) {
	// "it's fine" — single quote inside double-quoted string
	tokens, err := Tokenize(`msg="it's fine"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 || tokens[0] != `msg="it's fine"` {
		t.Errorf("unexpected token: %v", tokens)
	}
}

func TestTokenize_SingleQuoteContainingDoubleQuote(t *testing.T) {
	// 'say "hi"' — double quote inside single-quoted string
	tokens, err := Tokenize(`msg='say "hi"'`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 || tokens[0] != `msg='say "hi"'` {
		t.Errorf("unexpected token: %v", tokens)
	}
}

func TestTokenize_EmptyDoubleQuotedString(t *testing.T) {
	tokens, err := Tokenize(`msg=""`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 || tokens[0] != `msg=""` {
		t.Errorf("unexpected token: %v", tokens)
	}
}

func TestTokenize_ObjectValueWithSpaces(t *testing.T) {
	tokens, err := Tokenize("data={key: value with spaces}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 || tokens[0] != "data={key: value with spaces}" {
		t.Errorf("unexpected token: %v", tokens)
	}
}

func TestTokenize_BareObjectToken(t *testing.T) {
	tokens, err := Tokenize("{key: value}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 || tokens[0] != "{key: value}" {
		t.Errorf("unexpected token: %v", tokens)
	}
}

func TestTokenize_ObjectDoesNotRecurse(t *testing.T) {
	// Inside {}, only the first } closes it. The trailing } is a plain character
	// that appends to the same buffer, so the whole thing is one token.
	tokens, err := Tokenize("{outer: {inner}}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 || tokens[0] != "{outer: {inner}}" {
		t.Errorf("expected 1 token '{outer: {inner}}', got %v", tokens)
	}
}

func TestTokenize_ArrayValueWithSpaces(t *testing.T) {
	tokens, err := Tokenize("items=[a, b, c]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 || tokens[0] != "items=[a, b, c]" {
		t.Errorf("unexpected token: %v", tokens)
	}
}

func TestTokenize_BareArrayToken(t *testing.T) {
	tokens, err := Tokenize("[one two three]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 || tokens[0] != "[one two three]" {
		t.Errorf("unexpected token: %v", tokens)
	}
}

func TestTokenize_ArrayDoesNotRecurse(t *testing.T) {
	// Inside [], only the first ] closes it. The trailing ] is a plain character
	// that appends to the same buffer, so the whole thing is one token.
	tokens, err := Tokenize("[[a, b]]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 || tokens[0] != "[[a, b]]" {
		t.Errorf("expected 1 token '[[a, b]]', got %v", tokens)
	}
}

func TestTokenize_InlineCallNoArgs(t *testing.T) {
	tokens, err := Tokenize("(dialog.dir.open)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 || tokens[0] != "(dialog.dir.open)" {
		t.Errorf("unexpected token: %v", tokens)
	}
}

func TestTokenize_InlineCallWithArgs(t *testing.T) {
	tokens, err := Tokenize("(dialog.file_picker filters=[*.yaml])")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 || tokens[0] != "(dialog.file_picker filters=[*.yaml])" {
		t.Errorf("unexpected token: %v", tokens)
	}
}

func TestTokenize_KeyBoundInlineCall(t *testing.T) {
	tokens, err := Tokenize("path=(dialog.dir.open)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 || tokens[0] != "path=(dialog.dir.open)" {
		t.Errorf("unexpected token: %v", tokens)
	}
}

func TestTokenize_NestedInlineCall(t *testing.T) {
	tokens, err := Tokenize("(outer (inner))")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 || tokens[0] != "(outer (inner))" {
		t.Errorf("unexpected token: %v", tokens)
	}
}

func TestTokenize_DeeplyNestedInlineCall(t *testing.T) {
	tokens, err := Tokenize("(a (b (c)))")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 || tokens[0] != "(a (b (c)))" {
		t.Errorf("unexpected token: %v", tokens)
	}
}

func TestTokenize_InlineCallWithArrayInsideParens(t *testing.T) {
	// [ inside () is just a character — not a [] context
	tokens, err := Tokenize("path=(dialog.file_picker filters=[*.yaml])")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 {
		t.Errorf("expected 1 token, got %v", tokens)
	}
}

func TestTokenize_SpreadForm(t *testing.T) {
	tokens, err := Tokenize("SPORE.node.install (dialog.dir.open ext=.manifest.spore.yaml) ~h1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"SPORE.node.install", "(dialog.dir.open ext=.manifest.spore.yaml)", "~h1"}
	if len(tokens) != len(want) {
		t.Fatalf("expected %v, got %v", want, tokens)
	}
	for i, w := range want {
		if tokens[i] != w {
			t.Errorf("[%d] expected %q, got %q", i, w, tokens[i])
		}
	}
}

func TestTokenize_FullRealisticInstallMessage(t *testing.T) {
	tokens, err := Tokenize("SPORE.node.install path=(dialog.file_picker filters=[*.yaml]) ~h1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"SPORE.node.install", "path=(dialog.file_picker filters=[*.yaml])", "~h1"}
	if len(tokens) != len(want) {
		t.Fatalf("expected %v, got %v", want, tokens)
	}
	for i, w := range want {
		if tokens[i] != w {
			t.Errorf("[%d] expected %q, got %q", i, w, tokens[i])
		}
	}
}

func TestTokenize_ErrorResponseWithQuotedWhat(t *testing.T) {
	raw := `~h1:time.get_time error code=RouteNotFound what="No node connected for this subject" spore_error capture=SPORE.hub`
	tokens, err := Tokenize(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Ensure what= is a single token
	found := false
	for _, tok := range tokens {
		if tok == `what="No node connected for this subject"` {
			found = true
		}
	}
	if !found {
		t.Errorf("expected what= to be a single quoted token, got %v", tokens)
	}
}

func TestTokenize_ObjectWithInternalSpacesAmongOtherTokens(t *testing.T) {
	tokens, err := Tokenize("cmd data={key: value with spaces} name=test ~h1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"cmd", "data={key: value with spaces}", "name=test", "~h1"}
	if len(tokens) != len(want) {
		t.Fatalf("expected %v, got %v", want, tokens)
	}
	for i, w := range want {
		if tokens[i] != w {
			t.Errorf("[%d] expected %q, got %q", i, w, tokens[i])
		}
	}
}

func TestTokenize_QuotedStringAmongOtherTokens(t *testing.T) {
	tokens, err := Tokenize(`logger.append entry="started the process" ~l1`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"logger.append", `entry="started the process"`, "~l1"}
	if len(tokens) != len(want) {
		t.Fatalf("expected %v, got %v", want, tokens)
	}
	for i, w := range want {
		if tokens[i] != w {
			t.Errorf("[%d] expected %q, got %q", i, w, tokens[i])
		}
	}
}

// =============================================================================
// Tokenize — unhappy paths
// =============================================================================

func TestTokenize_UnterminatedDoubleQuote(t *testing.T) {
	_, err := Tokenize(`msg="hello`)
	if err == nil {
		t.Fatal("expected error for unterminated double-quoted string, got nil")
	}
}

func TestTokenize_UnterminatedSingleQuote(t *testing.T) {
	_, err := Tokenize("msg='hello")
	if err == nil {
		t.Fatal("expected error for unterminated single-quoted string, got nil")
	}
}

func TestTokenize_UnterminatedObject(t *testing.T) {
	_, err := Tokenize("data={key: value")
	if err == nil {
		t.Fatal("expected error for unterminated object {, got nil")
	}
}

func TestTokenize_UnterminatedArray(t *testing.T) {
	_, err := Tokenize("items=[a, b, c")
	if err == nil {
		t.Fatal("expected error for unterminated array [, got nil")
	}
}

func TestTokenize_UnterminatedInlineCall(t *testing.T) {
	_, err := Tokenize("path=(dialog.dir.open")
	if err == nil {
		t.Fatal("expected error for unterminated inline call (, got nil")
	}
}

func TestTokenize_UnterminatedOuterInlineCallWithClosedInner(t *testing.T) {
	// Inner closes correctly but outer does not
	_, err := Tokenize("path=(outer (inner)")
	if err == nil {
		t.Fatal("expected error for unterminated outer (, got nil")
	}
}

func TestTokenize_UnterminatedDoubleQuoteAtEndOfMessage(t *testing.T) {
	_, err := Tokenize(`cmd arg="unclosed`)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestTokenize_UnterminatedObjectAtEndOfMessage(t *testing.T) {
	_, err := Tokenize("cmd arg={unclosed")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// =============================================================================
// unquoteValue
// =============================================================================

func TestUnquoteValue_DoubleQuoted(t *testing.T) {
	if got := unquoteValue(`"hello world"`); got != "hello world" {
		t.Errorf("expected 'hello world', got %q", got)
	}
}

func TestUnquoteValue_SingleQuoted(t *testing.T) {
	if got := unquoteValue("'hello world'"); got != "hello world" {
		t.Errorf("expected 'hello world', got %q", got)
	}
}

func TestUnquoteValue_Unquoted(t *testing.T) {
	if got := unquoteValue("plainvalue"); got != "plainvalue" {
		t.Errorf("expected 'plainvalue', got %q", got)
	}
}

func TestUnquoteValue_ObjectNotStripped(t *testing.T) {
	if got := unquoteValue("{key: value}"); got != "{key: value}" {
		t.Errorf("expected object unchanged, got %q", got)
	}
}

func TestUnquoteValue_ArrayNotStripped(t *testing.T) {
	if got := unquoteValue("[a, b, c]"); got != "[a, b, c]" {
		t.Errorf("expected array unchanged, got %q", got)
	}
}

func TestUnquoteValue_EmptyDoubleQuoted(t *testing.T) {
	if got := unquoteValue(`""`); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestUnquoteValue_EmptySingleQuoted(t *testing.T) {
	if got := unquoteValue("''"); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestUnquoteValue_EmptyString(t *testing.T) {
	if got := unquoteValue(""); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestUnquoteValue_MismatchedQuotes(t *testing.T) {
	// Opening " with closing ' — should not be stripped
	if got := unquoteValue(`"hello'`); got != `"hello'` {
		t.Errorf("expected unchanged mismatched quotes, got %q", got)
	}
}

func TestUnquoteValue_EscapedNewline(t *testing.T) {
	if got := unquoteValue(`"line1\nline2"`); got != "line1\nline2" {
		t.Errorf("expected unescaped newline, got %q", got)
	}
}

func TestUnquoteValue_EscapedCarriageReturn(t *testing.T) {
	if got := unquoteValue(`"a\rb"`); got != "a\rb" {
		t.Errorf("expected unescaped CR, got %q", got)
	}
}

func TestUnquoteValue_EscapedTab(t *testing.T) {
	if got := unquoteValue(`"a\tb"`); got != "a\tb" {
		t.Errorf("expected unescaped tab, got %q", got)
	}
}

func TestUnquoteValue_EscapedBackslash(t *testing.T) {
	if got := unquoteValue(`"a\\b"`); got != `a\b` {
		t.Errorf("expected single backslash, got %q", got)
	}
}

func TestUnquoteValue_EscapedDoubleQuote(t *testing.T) {
	if got := unquoteValue(`"say \"hi\""`); got != `say "hi"` {
		t.Errorf("expected unescaped double quotes, got %q", got)
	}
}

func TestUnquoteValue_UnknownEscapePreserved(t *testing.T) {
	// \x is not a known escape — backslash and char are both preserved.
	if got := unquoteValue(`"a\xb"`); got != `a\xb` {
		t.Errorf("expected unknown escape preserved, got %q", got)
	}
}

func TestUnquoteValue_SingleQuotedNotUnescaped(t *testing.T) {
	// Backslash inside single-quoted string is not an escape character.
	if got := unquoteValue(`'a\nb'`); got != `a\nb` {
		t.Errorf("expected literal backslash-n, got %q", got)
	}
}

func TestUnquoteValue_MultilineYAML(t *testing.T) {
	// Simulate a YAML file content escaped for wire transport.
	escaped := `"id: aardvark\nname: Aardvark\n"`
	got := unquoteValue(escaped)
	want := "id: aardvark\nname: Aardvark\n"
	if got != want {
		t.Errorf("expected unescaped YAML content, got %q", got)
	}
}

// =============================================================================
// Tokenize — escape sequences in double-quoted strings
// =============================================================================

func TestTokenize_DoubleQuoteEscapedNewline(t *testing.T) {
	// \n inside "..." must be kept as a two-character escape sequence so the
	// token boundary is not disturbed by the literal newline.
	tokens, err := Tokenize(`content="line1\nline2"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token, got %v", tokens)
	}
	// The raw token retains the escape; unquoteValue decodes it.
	if got := unquoteValue(tokens[0][len("content="):]); got != "line1\nline2" {
		t.Errorf("expected decoded newline, got %q", got)
	}
}

func TestTokenize_DoubleQuoteEscapedQuote(t *testing.T) {
	tokens, err := Tokenize(`msg="say \"hi\""`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token, got %v", tokens)
	}
	if got := unquoteValue(tokens[0][len("msg="):]); got != `say "hi"` {
		t.Errorf("expected decoded double-quote, got %q", got)
	}
}

func TestTokenize_DoubleQuoteEscapedBackslash(t *testing.T) {
	tokens, err := Tokenize(`path="C:\\Users\\test"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token, got %v", tokens)
	}
	if got := unquoteValue(tokens[0][len("path="):]); got != `C:\Users\test` {
		t.Errorf("expected decoded backslashes, got %q", got)
	}
}

func TestTokenize_DoubleQuoteMultilineDoesNotSplit(t *testing.T) {
	// The tokenizer operates on a single-line string. Callers encode \n before
	// putting content on the wire; this test verifies the escape is preserved.
	tokens, err := Tokenize(`cmd content="first\nsecond\nthird" flag`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %v", tokens)
	}
	if tokens[1] != `content="first\nsecond\nthird"` {
		t.Errorf("unexpected content token: %q", tokens[1])
	}
}
