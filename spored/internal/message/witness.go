package message

import (
	"fmt"
	"spored/internal/utilities/out"
	"strings"
	"time"
)

type witness struct {
	raw string
	cast string
	witnessFlag string
}

// from spore
func Witness(body string, outs ...out.IOut) Message {
	witnessFlag := "spore_event"
	var builder strings.Builder
	builder.WriteString(body)
	for _, el := range outs {
		if el.String() == "spore_incoming" || el.String() == "spore_outgoing" || el.String() == "spore_node" {
			witnessFlag = el.String()
			continue
		}
		builder.WriteString(el.String())
	}
	return &witness{
		raw: builder.String(),
		cast: "n/a",
		witnessFlag: witnessFlag,
	}
}

// from node
func Node(raw string, cast string) Message {
	return &witness{
		raw: raw,
		cast: cast,
		witnessFlag: "spore_node",
	}
}

func (w *witness) Capability() string {
	return "n/a"
}

func (w *witness) Cast() string {
	return w.cast
}

func (w *witness) Capture() string {
	return "n/a"
}

func (w *witness) Arg(key string) (string, bool) {
	return "", false
}

func (w *witness) ArgIf(key string, ifnot string) string {
	return ifnot
}

func (w *witness) Flag(flag string) bool {
	return false
}

func (w *witness) Handle() string {
	return "n/a"
}

// func (w *witness) ToJSON() string {
// 	return w.raw
// }

func (w *witness) Wire() string {
	return "n/a"
}

// raw
// --> Could be anything.
// witness spore: +witness +message +spore_event +spore_time
// --> witness body="<raw>" spore_event spore_time=time
// witness node: +witness +body +cast +spore_node +spore_time
// --> witness body="<raw>" cast=witnesser.id spore_node spore_time=time 
func (w *witness) Witness() string {
	t := time.Now().UnixMilli()
	
	// spore_event
	if w.cast == "n/a" {
		return fmt.Sprintf(`witness body="%s" %s spore_time=%d`, w.raw, w.witnessFlag, t)	

	// spore_node
	} else {
		return fmt.Sprintf(`witness body="%s" cast=%s %s spore_time=%d`, w.raw, w.cast, w.witnessFlag, t)
	}
}