package message_event

type IBroadcastContext interface {
	Message() string
	Origin() string
	Topic() string
	Destination() string
}

type IBroadcastHandler interface {
	OnBroadcast(ctx IBroadcastContext)
	Ids() []string
}

type broadcast struct {
	message string
	origin string
	topic string
	destination string
}

func Broadcast(msg, origin, topic, dest string) IBroadcastContext {
	return &broadcast{
		message: msg,
		origin: origin,
		topic: topic,
		destination: dest,
	}
}

func (b *broadcast) Message() string { return b.message }
func (b *broadcast) Origin() string { return b.origin }
func (b *broadcast) Topic() string { return b.topic }
func (b *broadcast) Destination() string { return b.destination }