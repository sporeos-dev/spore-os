// Copyright 2026 mharr
// SPDX-License-Identifier: AGPL-3.0-only

package message

import "strings"

// ArgKind classifies a single parsed argument token.
type ArgKind int

const (
	// ArgKindHandle is a ~handle correlation token (e.g. ~h1).
	ArgKindHandle ArgKind = iota
	// ArgKindKeyValue is a plain key=value argument (e.g. path=/tmp, name="hello world").
	ArgKindKeyValue
	// ArgKindFlag is a bare flag token (e.g. recursive, verbose).
	ArgKindFlag
	// ArgKindKeyBind is a key-binding inline call: key=(subject [args...]).
	// The value side starts with ( and is tracked to the matching ).
	// Example: path=(dialog.file_picker filters=[*.yaml])
	ArgKindKeyBind
	// ArgKindSpread is a spread inline call: (subject [args...]) with no key.
	// All data fields from the inner response are merged into the outer call.
	// Example: (dialog.dir.open)
	ArgKindSpread
)

// Arg is a single structured token from a parsed message argument list.
type Arg struct {
	Kind ArgKind

	// Key holds:
	//   ArgKindKeyValue — the argument key name (e.g. "path")
	//   ArgKindKeyBind  — the argument key name (e.g. "path")
	//   ArgKindHandle   — the handle value without ~ (e.g. "h1")
	//   ArgKindFlag     — the flag name (e.g. "recursive")
	Key string

	// Value holds the raw value string for ArgKindKeyValue (may include
	// surrounding quotes, e.g. `"hello world"` or `[a, b, c]`).
	// Use unquoteValue to strip quote delimiters when interpreting the value.
	Value string

	// Inner holds the inner call string (parens stripped) for ArgKindKeyBind
	// and ArgKindSpread. This is the raw text of the command and its arguments
	// as they would appear in a standalone cast message.
	// Example: "dialog.file_picker filters=[*.yaml]"
	Inner string

	// Raw is the full token exactly as it appeared on the wire. Rebuild uses
	// this field to reconstruct the message string, so it must be kept in sync
	// if an arg is modified by the inline resolver.
	Raw string
}

// ParsedArgs is the structured form of a cast message's argument list.
// The first token (the command subject) is stored separately; all subsequent
// tokens are in Args.
type ParsedArgs struct {
	Command string
	Args    []Arg
}

// ParseArgs takes the token list produced by Tokenize and returns a ParsedArgs.
// The first token must be the command subject. Subsequent tokens are classified
// into Handle, KeyValue, Flag, KeyBind, or Spread args.
//
// Because Tokenize already groups key=(inner args) as a single token (tracking
// paren depth), the classification here is straightforward string inspection.
func ParseArgs(tokens []string) ParsedArgs {
	if len(tokens) == 0 {
		return ParsedArgs{}
	}
	out := ParsedArgs{
		Command: tokens[0],
		Args:    make([]Arg, 0, len(tokens)-1),
	}
	for _, tok := range tokens[1:] {
		out.Args = append(out.Args, ParseArgToken(tok))
	}
	return out
}

// ParseArgToken classifies a single token from a tokenised message argument
// list. The token must already be fully formed (e.g. parentheses balanced).
// This is exported so packages that process response messages (such as the
// router's inline resolver) can classify individual tokens.
func ParseArgToken(tok string) Arg {
	return parseArgToken(tok)
}

func parseArgToken(tok string) Arg {
	// Handle token: ~handle or ~handle:subject
	if strings.HasPrefix(tok, "~") {
		return Arg{Kind: ArgKindHandle, Key: strings.TrimPrefix(tok, "~"), Raw: tok}
	}

	// Spread inline: bare (subject [args...])
	if strings.HasPrefix(tok, "(") {
		// Strip surrounding parens to get the inner call string.
		inner := tok[1 : len(tok)-1]
		return Arg{Kind: ArgKindSpread, Inner: inner, Raw: tok}
	}

	// Key-value or key-binding: contains =
	if eq := strings.IndexByte(tok, '='); eq > 0 {
		key := tok[:eq]
		val := tok[eq+1:]
		if strings.HasPrefix(val, "(") {
			// Key-binding inline: key=(subject [args...])
			inner := val[1 : len(val)-1]
			return Arg{Kind: ArgKindKeyBind, Key: key, Inner: inner, Raw: tok}
		}
		return Arg{Kind: ArgKindKeyValue, Key: key, Value: val, Raw: tok}
	}

	// Bare flag
	return Arg{Kind: ArgKindFlag, Key: tok, Raw: tok}
}

// HasInlineForm reports whether the ParsedArgs contains any inline call slot
// — either a key-binding (key=(inner)) or a spread ((inner)). The hub uses
// this as the fast pre-routing check before deciding whether to enter the
// inline resolver.
func (p *ParsedArgs) HasInlineForm() bool {
	for _, arg := range p.Args {
		if arg.Kind == ArgKindKeyBind || arg.Kind == ArgKindSpread {
			return true
		}
	}
	return false
}

// SpreadCount returns the number of spread inline forms found in the arg list.
// Per the spec, at most one spread is allowed per outer call; callers should
// return an error to the caller if this returns > 1.
func (p *ParsedArgs) SpreadCount() int {
	n := 0
	for _, arg := range p.Args {
		if arg.Kind == ArgKindSpread {
			n++
		}
	}
	return n
}

// Rebuild reconstructs a raw message string from the ParsedArgs by joining the
// command and each Arg.Raw with spaces. The result is suitable for passing
// back into message.Parse for re-routing after inline substitution.
//
// Important: use Cast.ParsedArgs() (which operates on the pre-injection raw)
// rather than tokenising Cast.ToString(). That way Rebuild never includes the
// hub-injected cast= field, so it is re-injected naturally when the resolved
// message re-enters the router.
func (p *ParsedArgs) Rebuild() string {
	if len(p.Args) == 0 {
		return p.Command
	}
	parts := make([]string, 0, 1+len(p.Args))
	parts = append(parts, p.Command)
	for _, arg := range p.Args {
		parts = append(parts, arg.Raw)
	}
	return strings.Join(parts, " ")
}

// ReplaceArg replaces the arg at the given index with one or more new args.
// Used by the inline resolver to substitute a resolved key-bind or spread slot.
// For a key-bind (one→one), pass a single replacement arg.
// For a spread (one→many), pass all the spread data field args.
func (p *ParsedArgs) ReplaceArg(index int, replacements ...Arg) {
	before := p.Args[:index]
	after := p.Args[index+1:]
	newArgs := make([]Arg, 0, len(before)+len(replacements)+len(after))
	newArgs = append(newArgs, before...)
	newArgs = append(newArgs, replacements...)
	newArgs = append(newArgs, after...)
	p.Args = newArgs
}
