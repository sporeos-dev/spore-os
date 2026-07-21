package error

import (
	"fmt"
	"log/slog"
	"strings"
	"time"
)

type Error struct {
	Code Code
	Module Module
	What string
	Extra string
}

func New(code Code, module Module, what string, out ...fmt.Stringer) *Error {
	var extra strings.Builder
	for i, o := range out {
		if i > 0 {
			extra .WriteString(", ")
		}
		extra .WriteString(o.String())
	}
	err := &Error{
		Code: code,
		Module: module,
		What: what,
		Extra: extra.String(),
	}
	slog.Error(err.Error())
	return err
}

func (e *Error) Append(out fmt.Stringer) {
	if len(e.Extra) == 0 {
		e.Extra = out.String()
	} else {
		e.Extra = e.Extra + ", " + out.String()
	}
}

func (e *Error) Error() string {
	if len(e.Extra) > 0 {
		return fmt.Sprintf("[%s::in::%s] %s (%s)", e.Code, e.Module, e.What, e.Extra)
	} else {
		return fmt.Sprintf("[%s::in::%s] %s", e.Code, e.Module, e.What)
	}
}

func (e *Error) Wire() string {
	return fmt.Sprintf(`error code=%s module=%s what="%s" extra=[%s]`, e.Code, e.Module, e.What, e.Extra)
}

func (e *Error) Witness() string {
	t := time.Now().UnixMilli()
	return fmt.Sprintf(`witness error code=%s module=%s what="%s" extra=[%s] spore_error spore_time=%d`, e.Code, e.Module, e.What, e.Extra, t)
}
