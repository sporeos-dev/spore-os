// Copyright 2026 mharr
// SPDX-License-Identifier: AGPL-3.0-only

package message

import (
	"testing"
)

// ---------------------------------------------------------------------------
// parseArgToken
// ---------------------------------------------------------------------------

func TestParseArgToken_Handle(t *testing.T) {
	arg := parseArgToken("~h1")
	if arg.Kind != ArgKindHandle {
		t.Fatalf("expected ArgKindHandle, got %v", arg.Kind)
	}
	if arg.Key != "h1" {
		t.Fatalf("expected key=h1, got %q", arg.Key)
	}
	if arg.Raw != "~h1" {
		t.Fatalf("expected Raw=~h1, got %q", arg.Raw)
	}
}

func TestParseArgToken_HandleWithSubject(t *testing.T) {
	// ~handle:subject appears only on response messages, but ParseArgs must be
	// tolerant of it in case someone tokenises a response line.
	arg := parseArgToken("~h1:clock.get_time")
	if arg.Kind != ArgKindHandle {
		t.Fatalf("expected ArgKindHandle, got %v", arg.Kind)
	}
	if arg.Key != "h1:clock.get_time" {
		t.Fatalf("expected key=h1:clock.get_time, got %q", arg.Key)
	}
}

func TestParseArgToken_Flag(t *testing.T) {
	arg := parseArgToken("recursive")
	if arg.Kind != ArgKindFlag {
		t.Fatalf("expected ArgKindFlag, got %v", arg.Kind)
	}
	if arg.Key != "recursive" {
		t.Fatalf("expected key=recursive, got %q", arg.Key)
	}
	if arg.Raw != "recursive" {
		t.Fatalf("expected Raw=recursive, got %q", arg.Raw)
	}
}

func TestParseArgToken_KeyValue(t *testing.T) {
	arg := parseArgToken("path=/tmp")
	if arg.Kind != ArgKindKeyValue {
		t.Fatalf("expected ArgKindKeyValue, got %v", arg.Kind)
	}
	if arg.Key != "path" {
		t.Fatalf("expected key=path, got %q", arg.Key)
	}
	if arg.Value != "/tmp" {
		t.Fatalf("expected value=/tmp, got %q", arg.Value)
	}
}

func TestParseArgToken_KeyValueQuoted(t *testing.T) {
	arg := parseArgToken(`name="hello world"`)
	if arg.Kind != ArgKindKeyValue {
		t.Fatalf("expected ArgKindKeyValue, got %v", arg.Kind)
	}
	if arg.Key != "name" {
		t.Fatalf("expected key=name, got %q", arg.Key)
	}
	// Value retains the quotes — unquoteValue is a separate step.
	if arg.Value != `"hello world"` {
		t.Fatalf("expected value=%q, got %q", `"hello world"`, arg.Value)
	}
}

func TestParseArgToken_KeyValueArray(t *testing.T) {
	// The tokenizer already delivers [a, b, c] as a single value.
	arg := parseArgToken("filters=[*.yaml]")
	if arg.Kind != ArgKindKeyValue {
		t.Fatalf("expected ArgKindKeyValue, got %v", arg.Kind)
	}
	if arg.Key != "filters" {
		t.Fatalf("expected key=filters, got %q", arg.Key)
	}
	if arg.Value != "[*.yaml]" {
		t.Fatalf("expected value=[*.yaml], got %q", arg.Value)
	}
}

func TestParseArgToken_KeyBind(t *testing.T) {
	arg := parseArgToken("path=(dialog.file_picker)")
	if arg.Kind != ArgKindKeyBind {
		t.Fatalf("expected ArgKindKeyBind, got %v", arg.Kind)
	}
	if arg.Key != "path" {
		t.Fatalf("expected key=path, got %q", arg.Key)
	}
	if arg.Inner != "dialog.file_picker" {
		t.Fatalf("expected inner=dialog.file_picker, got %q", arg.Inner)
	}
	if arg.Raw != "path=(dialog.file_picker)" {
		t.Fatalf("expected Raw=path=(dialog.file_picker), got %q", arg.Raw)
	}
}

func TestParseArgToken_KeyBindWithInnerArgs(t *testing.T) {
	// Inner call has its own arguments including a bracket.
	// The tokenizer has already ensured this is a single token.
	arg := parseArgToken("path=(dialog.file_picker filters=[*.yaml])")
	if arg.Kind != ArgKindKeyBind {
		t.Fatalf("expected ArgKindKeyBind, got %v", arg.Kind)
	}
	if arg.Key != "path" {
		t.Fatalf("expected key=path, got %q", arg.Key)
	}
	if arg.Inner != "dialog.file_picker filters=[*.yaml]" {
		t.Fatalf("expected inner=%q, got %q", "dialog.file_picker filters=[*.yaml]", arg.Inner)
	}
}

func TestParseArgToken_Spread(t *testing.T) {
	arg := parseArgToken("(dialog.dir.open)")
	if arg.Kind != ArgKindSpread {
		t.Fatalf("expected ArgKindSpread, got %v", arg.Kind)
	}
	if arg.Inner != "dialog.dir.open" {
		t.Fatalf("expected inner=dialog.dir.open, got %q", arg.Inner)
	}
	if arg.Raw != "(dialog.dir.open)" {
		t.Fatalf("expected Raw=(dialog.dir.open), got %q", arg.Raw)
	}
}

func TestParseArgToken_SpreadWithInnerArgs(t *testing.T) {
	arg := parseArgToken("(dialog.dir.open filters=[*.yaml])")
	if arg.Kind != ArgKindSpread {
		t.Fatalf("expected ArgKindSpread, got %v", arg.Kind)
	}
	if arg.Inner != "dialog.dir.open filters=[*.yaml]" {
		t.Fatalf("expected inner=%q, got %q", "dialog.dir.open filters=[*.yaml]", arg.Inner)
	}
}

// ---------------------------------------------------------------------------
// ParseArgs (full message token list)
// ---------------------------------------------------------------------------

func TestParseArgs_Empty(t *testing.T) {
	p := ParseArgs(nil)
	if p.Command != "" {
		t.Fatalf("expected empty command, got %q", p.Command)
	}
	if len(p.Args) != 0 {
		t.Fatalf("expected no args, got %d", len(p.Args))
	}
}

func TestParseArgs_CommandOnly(t *testing.T) {
	p := ParseArgs([]string{"clock.get_time"})
	if p.Command != "clock.get_time" {
		t.Fatalf("expected command=clock.get_time, got %q", p.Command)
	}
	if len(p.Args) != 0 {
		t.Fatalf("expected no args, got %d", len(p.Args))
	}
}

func TestParseArgs_SimpleCall(t *testing.T) {
	// filesystem.read path=/documents/notes.txt ~r1
	tokens := []string{"filesystem.read", "path=/documents/notes.txt", "~r1"}
	p := ParseArgs(tokens)

	if p.Command != "filesystem.read" {
		t.Fatalf("wrong command: %q", p.Command)
	}
	if len(p.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(p.Args))
	}
	if p.Args[0].Kind != ArgKindKeyValue || p.Args[0].Key != "path" {
		t.Fatalf("arg[0] should be KeyValue path, got %+v", p.Args[0])
	}
	if p.Args[1].Kind != ArgKindHandle || p.Args[1].Key != "r1" {
		t.Fatalf("arg[1] should be Handle r1, got %+v", p.Args[1])
	}
}

func TestParseArgs_CallWithFlags(t *testing.T) {
	// filesystem.read path=/tmp recursive verbose ~r1
	tokens := []string{"filesystem.read", "path=/tmp", "recursive", "verbose", "~r1"}
	p := ParseArgs(tokens)

	if p.Args[0].Kind != ArgKindKeyValue {
		t.Fatalf("arg[0] should be KeyValue, got %v", p.Args[0].Kind)
	}
	if p.Args[1].Kind != ArgKindFlag || p.Args[1].Key != "recursive" {
		t.Fatalf("arg[1] should be Flag recursive, got %+v", p.Args[1])
	}
	if p.Args[2].Kind != ArgKindFlag || p.Args[2].Key != "verbose" {
		t.Fatalf("arg[2] should be Flag verbose, got %+v", p.Args[2])
	}
	if p.Args[3].Kind != ArgKindHandle {
		t.Fatalf("arg[3] should be Handle, got %v", p.Args[3].Kind)
	}
}

func TestParseArgs_KeyBindInline(t *testing.T) {
	// SPORE.node.install path=(dialog.file_picker) ~h1
	// The tokenizer turns path=(dialog.file_picker) into a single token.
	tokens := []string{"SPORE.node.install", "path=(dialog.file_picker)", "~h1"}
	p := ParseArgs(tokens)

	if p.Args[0].Kind != ArgKindKeyBind {
		t.Fatalf("arg[0] should be ArgKindKeyBind, got %v", p.Args[0].Kind)
	}
	if p.Args[0].Key != "path" {
		t.Fatalf("arg[0].Key should be path, got %q", p.Args[0].Key)
	}
	if p.Args[0].Inner != "dialog.file_picker" {
		t.Fatalf("arg[0].Inner should be dialog.file_picker, got %q", p.Args[0].Inner)
	}
}

func TestParseArgs_SpreadInline(t *testing.T) {
	// filesystem.list (dialog.dir.open) recursive ~h1
	tokens := []string{"filesystem.list", "(dialog.dir.open)", "recursive", "~h1"}
	p := ParseArgs(tokens)

	if p.Args[0].Kind != ArgKindSpread {
		t.Fatalf("arg[0] should be ArgKindSpread, got %v", p.Args[0].Kind)
	}
	if p.Args[0].Inner != "dialog.dir.open" {
		t.Fatalf("arg[0].Inner should be dialog.dir.open, got %q", p.Args[0].Inner)
	}
}

func TestParseArgs_FromTokenize(t *testing.T) {
	// Round-trip: Tokenize then ParseArgs on a message with bracket inside inline.
	raw := "SPORE.node.install path=(dialog.file_picker filters=[*.yaml]) ~h1"
	tokens, err := Tokenize(raw)
	if err != nil {
		t.Fatalf("Tokenize error: %v", err)
	}
	p := ParseArgs(tokens)

	if p.Command != "SPORE.node.install" {
		t.Fatalf("wrong command: %q", p.Command)
	}
	if len(p.Args) != 2 {
		t.Fatalf("expected 2 args (keybind + handle), got %d", len(p.Args))
	}
	if p.Args[0].Kind != ArgKindKeyBind {
		t.Fatalf("arg[0] should be ArgKindKeyBind, got %v", p.Args[0].Kind)
	}
	if p.Args[0].Inner != "dialog.file_picker filters=[*.yaml]" {
		t.Fatalf("arg[0].Inner wrong: %q", p.Args[0].Inner)
	}
}

// ---------------------------------------------------------------------------
// HasInlineForm
// ---------------------------------------------------------------------------

func TestHasInlineForm_NoInlines(t *testing.T) {
	tokens, _ := Tokenize("clock.get_time timezone=UTC ~h1")
	p := ParseArgs(tokens)
	if p.HasInlineForm() {
		t.Fatal("expected no inline forms")
	}
}

func TestHasInlineForm_KeyBind(t *testing.T) {
	tokens, _ := Tokenize("SPORE.node.install path=(dialog.file_picker) ~h1")
	p := ParseArgs(tokens)
	if !p.HasInlineForm() {
		t.Fatal("expected HasInlineForm to be true for key-bind")
	}
}

func TestHasInlineForm_Spread(t *testing.T) {
	tokens, _ := Tokenize("filesystem.list (dialog.dir.open) ~h1")
	p := ParseArgs(tokens)
	if !p.HasInlineForm() {
		t.Fatal("expected HasInlineForm to be true for spread")
	}
}

// ---------------------------------------------------------------------------
// SpreadCount
// ---------------------------------------------------------------------------

func TestSpreadCount_Zero(t *testing.T) {
	tokens, _ := Tokenize("filesystem.read path=/tmp ~h1")
	p := ParseArgs(tokens)
	if p.SpreadCount() != 0 {
		t.Fatalf("expected SpreadCount=0, got %d", p.SpreadCount())
	}
}

func TestSpreadCount_One(t *testing.T) {
	tokens, _ := Tokenize("filesystem.list (dialog.dir.open) recursive ~h1")
	p := ParseArgs(tokens)
	if p.SpreadCount() != 1 {
		t.Fatalf("expected SpreadCount=1, got %d", p.SpreadCount())
	}
}

func TestSpreadCount_Two(t *testing.T) {
	// Two spread inlines — invalid per spec, but SpreadCount must count them.
	// The resolver (not the parser) is responsible for rejecting this.
	tokens, _ := Tokenize("some.cmd (a.call) (b.call) ~h1")
	p := ParseArgs(tokens)
	if p.SpreadCount() != 2 {
		t.Fatalf("expected SpreadCount=2, got %d", p.SpreadCount())
	}
}

// ---------------------------------------------------------------------------
// Rebuild
// ---------------------------------------------------------------------------

func TestRebuild_CommandOnly(t *testing.T) {
	p := ParsedArgs{Command: "clock.get_time", Args: nil}
	if got := p.Rebuild(); got != "clock.get_time" {
		t.Fatalf("expected %q, got %q", "clock.get_time", got)
	}
}

func TestRebuild_SimpleCall(t *testing.T) {
	raw := "filesystem.read path=/tmp recursive ~r1"
	tokens, _ := Tokenize(raw)
	p := ParseArgs(tokens)
	if got := p.Rebuild(); got != raw {
		t.Fatalf("expected %q, got %q", raw, got)
	}
}

func TestRebuild_WithInline(t *testing.T) {
	raw := "SPORE.node.install path=(dialog.file_picker filters=[*.yaml]) ~h1"
	tokens, _ := Tokenize(raw)
	p := ParseArgs(tokens)
	if got := p.Rebuild(); got != raw {
		t.Fatalf("expected %q, got %q", raw, got)
	}
}

func TestRebuild_AfterReplaceKeyBind(t *testing.T) {
	// Simulate the inline resolver substituting path=(dialog.file_picker)
	// with the resolved value path=/chosen/path.
	raw := "SPORE.node.install path=(dialog.file_picker) ~h1"
	tokens, _ := Tokenize(raw)
	p := ParseArgs(tokens)

	// Find and replace the key-bind slot.
	for i, arg := range p.Args {
		if arg.Kind == ArgKindKeyBind && arg.Key == "path" {
			p.ReplaceArg(i, Arg{
				Kind:  ArgKindKeyValue,
				Key:   "path",
				Value: "/chosen/path",
				Raw:   "path=/chosen/path",
			})
			break
		}
	}

	want := "SPORE.node.install path=/chosen/path ~h1"
	if got := p.Rebuild(); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestRebuild_AfterReplaceSpread(t *testing.T) {
	// Simulate the inline resolver spreading two data fields into the outer call.
	raw := "filesystem.list (dialog.dir.open) recursive ~h1"
	tokens, _ := Tokenize(raw)
	p := ParseArgs(tokens)

	for i, arg := range p.Args {
		if arg.Kind == ArgKindSpread {
			p.ReplaceArg(i,
				Arg{Kind: ArgKindKeyValue, Key: "path", Value: "/projects", Raw: "path=/projects"},
				Arg{Kind: ArgKindKeyValue, Key: "filter", Value: "*.go", Raw: "filter=*.go"},
			)
			break
		}
	}

	want := "filesystem.list path=/projects filter=*.go recursive ~h1"
	if got := p.Rebuild(); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

// ---------------------------------------------------------------------------
// Cast.HasInlines / Cast.GetParsedArgs
// ---------------------------------------------------------------------------

func TestCast_HasInlines_False(t *testing.T) {
	c := &Cast{}
	if err := c.Parse("clock.get_time timezone=UTC ~h1", "com.example.agent"); err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if c.HasInlines() {
		t.Fatal("expected HasInlines=false for plain cast")
	}
}

func TestCast_HasInlines_TrueKeyBind(t *testing.T) {
	c := &Cast{}
	if err := c.Parse("SPORE.node.install path=(dialog.file_picker) ~h1", "com.example.agent"); err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if !c.HasInlines() {
		t.Fatal("expected HasInlines=true for key-bind cast")
	}
}

func TestCast_HasInlines_TrueSpread(t *testing.T) {
	c := &Cast{}
	if err := c.Parse("filesystem.list (dialog.dir.open) ~h1", "com.example.agent"); err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if !c.HasInlines() {
		t.Fatal("expected HasInlines=true for spread cast")
	}
}

func TestCast_GetParsedArgs_NoCastInjection(t *testing.T) {
	// GetParsedArgs must NOT include the hub-injected cast= field.
	c := &Cast{}
	if err := c.Parse("filesystem.read path=/tmp ~r1", "com.example.agent"); err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	p, err := c.GetParsedArgs()
	if err != nil {
		t.Fatalf("GetParsedArgs error: %v", err)
	}
	for _, arg := range p.Args {
		if arg.Kind == ArgKindKeyValue && arg.Key == "cast" {
			t.Fatal("GetParsedArgs should not include hub-injected cast= field")
		}
	}
	// Rebuild should match the original raw, not the injected form.
	want := "filesystem.read path=/tmp ~r1"
	if got := p.Rebuild(); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

// ---------------------------------------------------------------------------
// Cancelled — Parse and NewCancelled
// ---------------------------------------------------------------------------

func TestCancelled_Parse(t *testing.T) {
	c := &Cancelled{}
	if err := c.Parse("~h1:dialog.file_picker cancelled", "com.example.dialog"); err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if c.Handle() != "h1" {
		t.Fatalf("expected handle=h1, got %q", c.Handle())
	}
	if c.Command() != "dialog.file_picker" {
		t.Fatalf("expected command=dialog.file_picker, got %q", c.Command())
	}
	if c.Capture() != "com.example.dialog" {
		t.Fatalf("expected capture=com.example.dialog, got %q", c.Capture())
	}
	if !c.IsCancelled() {
		t.Fatal("expected IsCancelled=true")
	}
	if c.IsCapture() {
		t.Fatal("expected IsCapture=false")
	}
	// Injected capture= should appear in the wire string.
	if c.ToString() != "~h1:dialog.file_picker cancelled capture=com.example.dialog" {
		t.Fatalf("unexpected ToString: %q", c.ToString())
	}
}

func TestCancelled_ParseDoesNotInjectOk(t *testing.T) {
	c := &Cancelled{}
	_ = c.Parse("~h1:cmd cancelled", "com.example.node")
	raw := c.ToString()
	for _, tok := range splitSimple(raw) {
		if tok == "ok" {
			t.Fatal("cancelled response must not contain ok flag")
		}
	}
}

func TestNewCancelled(t *testing.T) {
	// Build a hub-generated cancelled from an original cast.
	orig := &Cast{}
	_ = orig.Parse("filesystem.list path=/tmp ~r1", "com.example.agent")

	c := NewCancelled(orig)
	if c.Handle() != "r1" {
		t.Fatalf("expected handle=r1, got %q", c.Handle())
	}
	if c.Command() != "filesystem.list" {
		t.Fatalf("expected command=filesystem.list, got %q", c.Command())
	}
	if c.Capture() != "SPORE.hub" {
		t.Fatalf("expected capture=SPORE.hub, got %q", c.Capture())
	}
	if !c.IsCancelled() {
		t.Fatal("expected IsCancelled=true")
	}
}

func TestCancelled_ParseDispatch(t *testing.T) {
	// End-to-end: message.Parse must dispatch ~...cancelled to *Cancelled.
	msg, err := Parse("~h1:cmd cancelled", "com.example.node")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if !msg.IsCancelled() {
		t.Fatalf("expected IsCancelled=true, got type that returns IsCancelled=false")
	}
}

func TestCapture_ParseNotDispatchedAsCancelled(t *testing.T) {
	// A plain capture (no cancelled token) must not dispatch to Cancelled.
	msg, err := Parse(`~h1:clock.get_time time="12:00" ok capture=com.example.clock`, "com.example.clock")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if msg.IsCancelled() {
		t.Fatal("plain capture must not be IsCancelled")
	}
	if !msg.IsCapture() {
		t.Fatal("plain capture must be IsCapture")
	}
}

// splitSimple splits on spaces — a lightweight helper for test assertions
// that doesn't need full tokenize semantics.
func splitSimple(s string) []string {
	var out []string
	for _, f := range []rune(s) {
		_ = f
	}
	// Use strings.Fields via the standard approach since we're in the message package.
	cur := ""
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
		} else {
			cur += string(s[i])
		}
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
