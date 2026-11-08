package witness_event

import (
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"time"
)

type WType string
const (
	WTypeIncoming WType = "spore.incoming"
	WTypeOutgoing WType = "spore.outgoing"
	WTypeSpore WType = "spore.event"
	WTypeNode WType = "spore.node"
	WTypeError WType = "spore.error"
)

type IWitnessContext interface {
	Message() string
	Origin() string
	Time() int64
	Type() WType
}

type IWitnessHandler interface {
	OnWitness(IWitnessContext)
}

type witness struct {
	message string
	origin string
	time int64
	wtype WType
}

func Incoming(msg, orig string) IWitnessContext {
	return &witness{
		message: msg, 
		origin: orig, 
		time: time.Now().UnixMilli(), 
		wtype: WTypeIncoming,
	}
}

func Outgoing(msg, orig string) IWitnessContext {
	return &witness{
		message: msg, 
		origin: orig, 
		time: time.Now().UnixMilli(), 
		wtype: WTypeOutgoing,
	}
}

func Spore(msg string) IWitnessContext {
	return &witness{
		message: msg, 
		origin: "dev.sporeos.SPORE",
		time: time.Now().UnixMilli(), 
		wtype: WTypeSpore,
	}
}

func Error(code error.Code, module error.Module, what string, outs []out.IOut) IWitnessContext {
	msg := error.Stringify(code, module, what, outs)
	return &witness{
		message: msg,
		origin: "dev.sporeos.SPORE",
		time: time.Now().UnixMilli(),
		wtype: WTypeError,
	}
}

func Node(msg, orig string) IWitnessContext {
	return &witness{
		message: msg,
		origin: orig,
		time: time.Now().UnixMilli(),
		wtype: WTypeNode,
	}
}

func (w *witness) Message() string {
	return w.message
}

func (w *witness) Origin() string {
	return w.origin
}

func (w *witness) Time() int64 {
	return w.time
}

func (w *witness) Type() WType {
	return w.wtype
}
