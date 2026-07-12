package error

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

