package message

import (
	"fmt"
	"spored/internal/iface"
	"spored/internal/utilities/out"
	"strings"
	"time"
)

type spore struct {
	request iface.Message
	responses []string
}

func Spore(request iface.Message, responses ...out.IOut) iface.Message {
	s := &spore {
		request: request,
		responses: make([]string, 0),
	}
	for _, el := range responses {
		s.responses = append(s.responses, el.String())
	}
	return s
}

func (s *spore) Capability() string {
	return s.request.Capability()
}

func (s *spore) Cast() string {
	return s.request.Cast()
}

func (s *spore) Capture() string {
	return "dev.sporeos.SPORE"
}

func (s *spore) Arg(key string) (string, bool) {
	return "", false
}

func (s *spore) ArgIf(key string, ifnot string) string {
	return ifnot
}

func (s *spore) Flag(flag string) bool {
	return false
}

func (s *spore) Handle() string {
	return s.request.Handle()
}

// raw
// --> {{ nothing in particular }}
// wire: +handle +subject +responses +ok +cast +capture
// --> ~handle:subject <responses> cast=requester.id capture=spore.id
func (s *spore) Wire() string {
	return fmt.Sprintf("~%s:%s %s ok cast=%s capture=%s", s.Handle(), s.Capability(), s.responsesToString(), s.Cast(), s.Capture())
}

// witness out: +witness +spore_outgoing +spore_time
// --> witness <wire> spore_outgoing spore_time=time
func (s *spore) Witness() string {
	t := time.Now().UnixMilli()
	return fmt.Sprintf("witness body='%s' spore_outgoing spore_time=%d", s.Wire(), t)
}

// 
// private
// helpers
//

func (s *spore) responsesToString() string {
	var res strings.Builder
	for i, el := range s.responses {
		if i == 0 {
			res.WriteString(el)
		} else {
			res.WriteString(" " + el)
		}
	}
	return res.String()
}