// spored/internal/spore/node.go

package spore

import (
    "fmt"
)

type NodeCommand struct {
    // fields...
}

func newNodeCommand() *NodeCommand {
    return &NodeCommand{}
}

func (n *NodeCommand) Handle(args []string) error {
    // logic for node...
    return nil
}
