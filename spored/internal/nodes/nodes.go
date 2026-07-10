package nodes

type Nodes struct {
	registry *registry
}

func New() *Nodes {
	nodes := &Nodes {
		registry: newRegistry(),
	}

	err := nodes.registry.loadRegistry() 
}

func (n *Nodes) Close() {
	n.registry.close()
}
