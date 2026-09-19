package error

import (
	"fmt"
	"log/slog"
	"spored/internal/iface"
	"strings"
	"time"
)

type Error struct {
	code Code
	module Module
	what string
	extras []string

	message iface.Message
	incomingSent bool
}

func New(code Code, module Module, what string, out ...fmt.Stringer) *Error {
    extraSlice := make([]string, len(out))
    for i, o := range out {
        extraSlice[i] = o.String()
    }
    
    err := &Error{
        code:   code,
        module: module,
        what:   what,
        extras:  extraSlice,
		message: nil,
		incomingSent: false,
    }

	slog.Error(err.Error())
    return err
}

func (e *Error) WithMessage(message iface.Message) *Error {
	e.message = message
	return e
}

func (e *Error) Append(out fmt.Stringer) {
	e.extras = append(e.extras, out.String())
}

//
// error
// interface
//

func (e *Error) Error() string{
	return fmt.Sprintf(`%s code=%s.%s %s`, e.what, e.code, e.module, e.extrasToString())
}

//
//
// iface.Message
// interface
//

func (e *Error) Capability() string {
	if e.message == nil {
		return "n/a"
	}
	return e.message.Capability()
}

func (e *Error) Cast() string {
	if e.message == nil {
		return "n/a"
	}
	return e.message.Cast()
}

func (e *Error) Capture() string {
	if e.message == nil {
		return "n/a"
	}
	return e.message.Capture()
}

func (e *Error) Arg(key string) (string, bool) {
	if e.message == nil {
		return "", false
	}
	return e.message.Arg(key)
}

func (e *Error) ArgIf(key string, ifnot string) string {
	if e.message == nil {
		return ifnot
	}
	return e.message.ArgIf(key, ifnot)
}

func (e *Error) Flag(flag string) bool {
	if e.message == nil {
		return false
	}
	return e.message.Flag(flag)
}

func (e *Error) Handle() string {
	if e.message == nil {
		return "n/a"
	}
	return e.message.Handle()
}

// build a response && an error
// ~handle:subject error code=code.module what="what" <extras> cast=cast (capture=capture)
func (e *Error) Wire() string {
	// no message
	// spore error
	// no destination
	if e.message == nil {
		return "n/a"
	}

	// capture is n/a
	// cast error
	// send back
	wire := fmt.Sprintf(`~%s:%s error code=%s.%s what="%s" %s cast=%s`, e.Handle(), e.Capability(), e.code, e.module, e.what, e.extrasToString(), e.Cast()) 

	if e.Capture() == "n/a" {
		return wire

	// otherwise
	// capture error
	// send forward
	} else {
		return fmt.Sprintf("%s capture=%s", wire, e.message.Capture())
	}
}

func (e *Error) Witness() string {
	t := time.Now().UnixMilli()

	// no message
	// spore error
	if e.message == nil {
		return fmt.Sprintf("witness body='error code=%s.%s what=\"%s\"' %s spore_event spore_time=%d", e.code, e.module, e.what, e.extrasToString(), t)

	// otherwise
	// outgoing
	} else {
		return fmt.Sprintf("witness body='%s' spore_outgoing spore_time=%d", e.Wire(), t)
	}
}

// 
// private
// helpers
//

func (e *Error) extrasToString() string {
	var ex strings.Builder
	for i, el := range e.extras {
		if i == 0 {
			ex.WriteString(el)
		} else {
			ex.WriteString(" " + el)
		}
	}
	return ex.String()
}
