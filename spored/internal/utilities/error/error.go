package error

import "fmt"

type Error struct {
	Code Code
	What string
}

func New(code Code, what string) *Error {
	return &Error{
		Code: code,
		What: what,
	}
}

func (e *Error) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.What)
}

