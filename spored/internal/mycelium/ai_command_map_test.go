package mycelium

import (
	"reflect"
	"spored/internal/event"
	manifest_event "spored/internal/event/manifest"
	"spored/internal/manifest"
	"testing"
)

func TestManifestWithdrawal(t *testing.T) {
	event.Init()
	commands := newCommandMap()
	event.Subscribe(commands)
	defer event.Unsubscribe(commands)

	first := &manifest.Manifest{
		ID:  "r",
		Api: []manifest.Command{{Name: "x.y"}, {Name: "read"}},
	}

	second := &manifest.Manifest{
		ID:  "p.r",
		Api: []manifest.Command{{Name: "x.y"}},
	}
	firstCtx := manifest_event.New(first)
	secondCtx := manifest_event.New(second)

	event.Dispatch().ManifestAnnouncement.Notify(firstCtx)
	event.Dispatch().ManifestAnnouncement.Notify(firstCtx)
	event.Dispatch().ManifestAnnouncement.Notify(secondCtx)
	event.Dispatch().ManifestWithdrawal.Notify(firstCtx)

	wantCommands := map[string]string{"p.r.x.y": "p.r"}
	wantAbbreviations := map[string][]string{
		"y":       {"p.r.x.y"},
		"x.y":     {"p.r.x.y"},
		"r.x.y":   {"p.r.x.y"},
		"p.r.x.y": {"p.r.x.y"},
	}
	if !reflect.DeepEqual(commands.commands, wantCommands) {
		t.Errorf("commands = %v; want %v", commands.commands, wantCommands)
	}
	if !reflect.DeepEqual(commands.abbreviations, wantAbbreviations) {
		t.Errorf("abbreviations = %v; want %v", commands.abbreviations, wantAbbreviations)
	}
	if node, ok := commands.retrieveNode("r.x.y"); !ok || node != second.ID {
		t.Errorf("retrieveNode(r.x.y) = %q, %t; want %q, true", node, ok, second.ID)
	}

	event.Dispatch().ManifestWithdrawal.Notify(firstCtx)
	if !reflect.DeepEqual(commands.commands, wantCommands) || !reflect.DeepEqual(commands.abbreviations, wantAbbreviations) {
		t.Fatalf("repeated withdrawal changed remaining entries: commands=%v abbreviations=%v", commands.commands, commands.abbreviations)
	}

	event.Unsubscribe(commands)
	event.Dispatch().ManifestWithdrawal.Notify(secondCtx)
	if !reflect.DeepEqual(commands.commands, wantCommands) || !reflect.DeepEqual(commands.abbreviations, wantAbbreviations) {
		t.Fatal("withdrawal reached the unsubscribed command map")
	}

	event.Subscribe(commands)
	event.Dispatch().ManifestWithdrawal.Notify(secondCtx)
	if len(commands.commands) != 0 || len(commands.abbreviations) != 0 {
		t.Errorf("final withdrawal left entries: commands=%v abbreviations=%v", commands.commands, commands.abbreviations)
	}
	if node, ok := commands.retrieveNode("y"); ok {
		t.Errorf("retrieveNode(y) = %q, true after withdrawal; want failure", node)
	}
}

func TestManifestWithdrawalDoesNotRemoveAnotherOwner(t *testing.T) {
	commands := newCommandMap()
	owner := &manifest.Manifest{ID: "r.x", Api: []manifest.Command{{Name: "y"}}}
	other := &manifest.Manifest{ID: "r", Api: []manifest.Command{{Name: "x.y"}}}

	commands.OnManifestAnnouncement(manifest_event.New(owner))
	commands.OnManifestWithdrawal(manifest_event.New(other))

	if node, ok := commands.commands["r.x.y"]; !ok || node != owner.ID {
		t.Errorf("command owner = %q, present = %t; want %q", node, ok, owner.ID)
	}
	if got := commands.abbreviations["y"]; !reflect.DeepEqual(got, []string{"r.x.y"}) {
		t.Errorf("abbreviation candidates = %v; want [r.x.y]", got)
	}
}
