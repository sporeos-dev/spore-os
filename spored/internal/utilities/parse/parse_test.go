package parse

import (
	"testing"
)

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
