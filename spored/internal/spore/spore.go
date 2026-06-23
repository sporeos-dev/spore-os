// spored/internal/spore/spore.go

package spore

import (
    "fmt"
)

type Spore struct {
    HelpCommand  *HelpCommand
    NodeCommand  *NodeCommand
    CommandCommand *CommandCommand
    ErrorCommand *ErrorCommand
}

func (s *Spore) Command(raw string, from string) error {
    parts := strings.Fields(raw)
    if len(parts) == 0 {
        return fmt.Errorf("empty command")
    }

    switch parts[0] {
    case "SPORE.help":
        return s.HelpCommand.Handle(parts[1:])
    case "node":
        return s.NodeCommand.Handle(parts[1:])
    case "command":
        return s.CommandCommand.Handle(parts[1:])
    case "error":
        return s.ErrorCommand.Handle(parts[1:])
    default:
        return fmt.Errorf("unknown command: %s", parts[0])
    }
}

func NewSpore() *Spore {
    return &Spore{
        HelpCommand:  newHelpCommand(),
        NodeCommand:  newNodeCommand(),
        CommandCommand: newCommandCommand(),
        ErrorCommand:  newErrorCommand(),
    }
}
