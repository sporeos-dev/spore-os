package parse

import (
	"testing"
)

// --- Tokenize ---

func TestTokenize_Simple(t *testing.T) {
	tokens, err := Tokenize("clock.get_time timezone=UTC ~h1")
	if err != "" {
		t.Fatalf("unexpected error: %s", err)
	}
	want := []string{"clock.get_time", "timezone=UTC", "~h1"}
	if len(tokens) != len(want) {
		t.Fatalf("got %v, want %v", tokens, want)
	}
	for i := range want {
		if tokens[i] != want[i] {
			t.Errorf("token[%d]: got %q, want %q", i, tokens[i], want[i])
		}
	}
}

func TestTokenize_QuotedValue(t *testing.T) {
	tokens, err := Tokenize(`logger.append entry="hello world" ~l1`)
	if err != "" {
		t.Fatalf("unexpected error: %s", err)
	}
	want := []string{"logger.append", `entry="hello world"`, "~l1"}
	for i := range want {
		if tokens[i] != want[i] {
			t.Errorf("token[%d]: got %q, want %q", i, tokens[i], want[i])
		}
	}
}

func TestTokenize_SingleQuotedValue(t *testing.T) {
	tokens, err := Tokenize(`filesystem.read path='/tmp/my file' recursive ~r1`)
	if err != "" {
		t.Fatalf("unexpected error: %s", err)
	}
	want := []string{"filesystem.read", "path='/tmp/my file'", "recursive", "~r1"}
	for i := range want {
		if tokens[i] != want[i] {
			t.Errorf("token[%d]: got %q, want %q", i, tokens[i], want[i])
		}
	}
}

func TestTokenize_ArrayArg(t *testing.T) {
	tokens, err := Tokenize("SPORE.node.list filters=[a, b, c] ~m1")
	if err != "" {
		t.Fatalf("unexpected error: %s", err)
	}
	want := []string{"SPORE.node.list", "filters=[a, b, c]", "~m1"}
	for i := range want {
		if tokens[i] != want[i] {
			t.Errorf("token[%d]: got %q, want %q", i, tokens[i], want[i])
		}
	}
}

func TestTokenize_InlineCall(t *testing.T) {
	tokens, err := Tokenize("SPORE.node.install path=(dialog.file_picker filters=[*.yaml]) ~i1")
	if err != "" {
		t.Fatalf("unexpected error: %s", err)
	}
	want := []string{"SPORE.node.install", "path=(dialog.file_picker filters=[*.yaml])", "~i1"}
	for i := range want {
		if tokens[i] != want[i] {
			t.Errorf("token[%d]: got %q, want %q", i, tokens[i], want[i])
		}
	}
}

func TestTokenize_UnterminatedDouble(t *testing.T) {
	_, err := Tokenize(`bad="unclosed`)
	if err == "" {
		t.Fatal("expected error for unterminated double-quoted string")
	}
}

func TestTokenize_UnterminatedSingle(t *testing.T) {
	_, err := Tokenize(`bad='unclosed`)
	if err == "" {
		t.Fatal("expected error for unterminated single-quoted string")
	}
}

// --- UnquoteValue ---

func TestUnquoteValue_Double(t *testing.T) {
	if got := UnquoteValue(`"hello world"`); got != "hello world" {
		t.Errorf("got %q", got)
	}
}

func TestUnquoteValue_Single(t *testing.T) {
	if got := UnquoteValue("'hello world'"); got != "hello world" {
		t.Errorf("got %q", got)
	}
}

func TestUnquoteValue_Plain(t *testing.T) {
	if got := UnquoteValue("plain"); got != "plain" {
		t.Errorf("got %q", got)
	}
}

func TestUnquoteValue_Escapes(t *testing.T) {
	if got := UnquoteValue(`"say \"hi\""`); got != `say "hi"` {
		t.Errorf("got %q", got)
	}
}

func TestUnquoteValue_NewlineEscape(t *testing.T) {
	if got := UnquoteValue(`"line1\nline2"`); got != "line1\nline2" {
		t.Errorf("got %q", got)
	}
}

// --- ArrayToStrings ---

func TestArrayToStrings_QuotedWithCommas(t *testing.T) {
	got := ArrayToStrings(`["AAA","BBB","CCC"]`)
	want := []string{"AAA", "BBB", "CCC"}
	assertStringSlice(t, got, want)
}

func TestArrayToStrings_QuotedNoCommas(t *testing.T) {
	got := ArrayToStrings(`["AAA" "BBB" "CCC"]`)
	want := []string{"AAA", "BBB", "CCC"}
	assertStringSlice(t, got, want)
}

func TestArrayToStrings_UnquotedNoCommas(t *testing.T) {
	got := ArrayToStrings(`[AAA BBB CCC]`)
	want := []string{"AAA", "BBB", "CCC"}
	assertStringSlice(t, got, want)
}

func TestArrayToStrings_UnquotedWithCommas(t *testing.T) {
	got := ArrayToStrings(`[AAA, BBB, CCC]`)
	want := []string{"AAA", "BBB", "CCC"}
	assertStringSlice(t, got, want)
}

func TestArrayToStrings_QuotedSpacesInValue(t *testing.T) {
	got := ArrayToStrings(`["hello world" "foo bar"]`)
	want := []string{"hello world", "foo bar"}
	assertStringSlice(t, got, want)
}

func assertStringSlice(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len: got %d (%v), want %d (%v)", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}
