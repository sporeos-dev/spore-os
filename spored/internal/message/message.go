package message

type Message interface{

	Witness() string

	// pub/sub
	Topic() string
}

func New() Message {

}

