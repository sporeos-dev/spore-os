package message_event

type IMessageContext interface {
	Message() string
	Origin() string
}

type IMessageHandler interface {
	OnMessage(ctx IMessageContext)
}

type message struct {
	message string
	origin string
}

func Raw(msg, org string) IMessageContext {
	return &message{
		message: msg,
		origin: org,
	}
}

func (m *message) Message() string {
	return m.message
}

func (m *message) Origin() string {
	return m.origin
}
