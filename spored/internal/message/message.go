package message

type Message struct {}

func New() *Message {
	return &Message{}
}

func (m *Message) Close() {}
