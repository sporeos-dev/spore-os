// spored/internal/spore/command.go

package spore

import (
    "fmt"
)

type CommandCommand struct {
    // fields...
}

func newCommandCommand() *CommandCommand {
    return &CommandCommand{}
}

func (c *CommandCommand) Handle(args []string) error {
    // logic for command...
    return nil
}
