// spored/internal/spore/help.go

package spore

import (
    "fmt"
)

type HelpCommand struct {
    // fields...
}

func newHelpCommand() *HelpCommand {
    return &HelpCommand{}
}

func (h *HelpCommand) Handle(args []string) error {
    // logic for help...
    return nil
}
