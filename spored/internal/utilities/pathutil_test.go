package utilities

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveAppPath_Empty(t *testing.T) {
	got, err := ResolveAppPath("", "/some/dir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestResolveAppPath_NA(t *testing.T) {
	got, err := ResolveAppPath("n/a", "/some/dir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "n/a" {
		t.Errorf("expected \"n/a\", got %q", got)
	}
}

func TestResolveAppPath_Absolute(t *testing.T) {
	abs := "/usr/local/bin/myapp"
	got, err := ResolveAppPath(abs, "/some/dir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != abs {
		t.Errorf("expected %q, got %q", abs, got)
	}
}

func TestResolveAppPath_Relative(t *testing.T) {
	base := "/srv/spore/node-a"
	got, err := ResolveAppPath("node-a", base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(base, "node-a")
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestResolveAppPath_RelativeDotSlash(t *testing.T) {
	base := "/srv/spore/node-a"
	got, err := ResolveAppPath("./node-a", base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(base, "node-a")
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestResolveAppPath_RelativeDotDot(t *testing.T) {
	base := "/srv/spore/node-a"
	got, err := ResolveAppPath("../bin/node-a", base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(base, "../bin/node-a")
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestResolveAppPath_Tilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir available")
	}
	got, err := ResolveAppPath("~/bin/myapp", "/some/dir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(home, "bin/myapp")
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
	if !strings.HasPrefix(got, home) {
		t.Errorf("expected result to begin with home dir %q, got %q", home, got)
	}
}
