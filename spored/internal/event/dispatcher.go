package event

import (
	"spored/internal/base"
	witness_event "spored/internal/event/witness"
)

type Dispatcher struct {
	Witness *SimpleEvent[witness_event.IWitnessHandler, witness_event.IWitnessContext]
}

func New() *Dispatcher {
	return &Dispatcher{
		Witness: Simple(
			func(sub witness_event.IWitnessHandler, ctx witness_event.IWitnessContext) { 
				sub.OnWitness(ctx)
			},
		),
	}
}

func (d *Dispatcher) Sub(base base.IBase) {
	d.Witness.Sub(base)
}

func (d *Dispatcher) Unsub(base base.IBase) {
	d.Witness.Unsub(base)
}