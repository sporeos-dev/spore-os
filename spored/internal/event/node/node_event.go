package node_event

type INodeConnectionHandler interface {
	OnNodeConnection(id string)
}

type INodeDisconnectionHandler interface {
	OnNodeDisconnection(id string)
}
