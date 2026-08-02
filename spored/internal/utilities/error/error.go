package error

import (
	"fmt"
	"log/slog"
	"spored/internal/message"
	"strings"
	"time"
)

type Error struct {
	Code Code
	Module Module
	What string
	Extra []string
}

func New(code Code, module Module, what string, out ...fmt.Stringer) *Error {
    extraSlice := make([]string, len(out))
    for i, o := range out {
        extraSlice[i] = o.String()
    }
    
    err := &Error{
        Code:   code,
        Module: module,
        What:   what,
        Extra:  extraSlice,
    }
    slog.Error(err.error())
    return err
}

func (e *Error) Append(out fmt.Stringer) {
	e.Extra = append(e.Extra, out.String())
}

func (e *Error) error() string {
	if len(e.Extra) > 0 {
		return fmt.Sprintf("[%s::in::%s] %s (%s)", e.Code, e.Module, e.What, e.Extra)
	} else {
		return fmt.Sprintf("[%s::in::%s] %s", e.Code, e.Module, e.What)
	}
}

func (e *Error) Wire(handle *string, subject *string, cast *string, capture *string) string {
	var sb strings.Builder
	if handle != nil && subject != nil {
		sb.WriteString(fmt.Sprintf(`~%s:%s `, *handle, *subject))
	}
	sb.WriteString(fmt.Sprintf(`error code=%s.%s what="%s"`, e.Code, e.Module, e.What))
	for _, el := range e.Extra {
		sb.WriteString(fmt.Sprintf(` %s`, el))
	}
	if cast != nil {
		sb.WriteString(fmt.Sprintf(` cast=%s`, *cast))
	}
	if capture != nil {
		sb.WriteString(fmt.Sprintf(` capture=%s`, *capture))
	}
	return sb.String()
}

func (e *Error) Witness(wtype message.WitnessType) string {
	t := time.Now().UnixMilli()
	var sb strings.Builder
	for _, el := range e.Extra {
		sb.WriteString(fmt.Sprintf(` %s`, el))
	}
	return fmt.Sprintf(`witness %s error code=%s.%s what="%s"%s spore_error spore_time=%d`, wtype, e.Code, e.Module, e.What, sb.String(), t)
}
