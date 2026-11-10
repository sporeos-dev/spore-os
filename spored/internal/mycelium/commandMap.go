package mycelium

import (
	"slices"
	"spored/internal/base"
	manifest_event "spored/internal/event/manifest"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"spored/internal/witness"
	"strings"
	"sync"
)

type commandMap struct {
	*base.Base
	mu            sync.RWMutex
	commands      map[string]string   // cmd -> node
	abbreviations map[string][]string // [abbr] -> cmd; multiple implies collision
}

func newCommandMap() *commandMap {
	return &commandMap{
		Base:          base.New().Set("mycelium", "command-map"),
		commands:      make(map[string]string),
		abbreviations: make(map[string][]string),
	}
}

func (c *commandMap) mapCommand(cmd, node string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.commands[cmd] = node
}

func (c *commandMap) retrieveNode(cmd string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	arr, ok := c.abbreviations[cmd]
	if !ok || len(arr) == 0 {
		witness.Error(error.Missing, error.Mycelium, "command not found")
		return "", false
	} else if len(arr) > 1 {
		witness.Error(error.Collision, error.Mycelium, "command collision", out.Pair("command", cmd))
		return "", false
	}

	node, ok := c.commands[arr[0]]
	if !ok {
		witness.Error(error.Missing, error.Mycelium, "command not found")
		return "", false
	}
	return node, true
}

// manifest announcement
func (c *commandMap) OnManifestAnnouncement(ctx manifest_event.IManifestAnnouncementContext) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, cmd := range ctx.Manifest().Api {
		fqName := ctx.Manifest().ID + "." + cmd.Name
		for _, abbr := range commandAbbreviations(fqName) {
			if !slices.Contains(c.abbreviations[abbr], fqName) {
				c.abbreviations[abbr] = append(c.abbreviations[abbr], fqName)
			}
		}
		c.commands[fqName] = ctx.Manifest().ID
	}
}

// manifest withdrawal
func (c *commandMap) OnManifestWithdrawal(ctx manifest_event.IManifestWithdrawalContext) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, cmd := range ctx.Manifest().Api {
		fqName := ctx.Manifest().ID + "." + cmd.Name
		node, ok := c.commands[fqName]
		if !ok || node != ctx.Manifest().ID {
			continue
		}
		delete(c.commands, fqName)
		for _, abbr := range commandAbbreviations(fqName) {
			remaining := slices.DeleteFunc(c.abbreviations[abbr], func(candidate string) bool {
				return candidate == fqName
			})
			if len(remaining) == 0 {
				delete(c.abbreviations, abbr)
			} else {
				c.abbreviations[abbr] = remaining
			}
		}
	}
}

//
//
// private
// helpers
//

func commandAbbreviations(fqName string) []string {
	parts := strings.Split(fqName, ".")
	abbreviations := make([]string, 0, len(parts))
	for i := len(parts) - 1; i >= 0; i-- {
		abbreviations = append(abbreviations, strings.Join(parts[i:], "."))
	}
	return abbreviations
}
