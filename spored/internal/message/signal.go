package message

type signal struct {
	handle string
}

func Signal(handle string) *signal {
	return &signal{
		handle: handle,
	}
}

func (s *signal) Capability() string {
	return ""
}

func (s *signal) Cast() string {
	return "dev.sporeos.SPORE"
}

func (s *signal) Capture() string {
	return "dev.sporeos.SPORE"
}

func (s *signal) Arg(key string) (string, bool) {
	return "", false
}

func (s *signal) ArgIf(key string, ifnot string) string {
	return ifnot
}

func (s *signal) Flag(flag string) bool {
	return false
}

func (s *signal) Handle() string {
	return s.handle
}

func (s *signal) Wire() string {
	return s.handle
}

func (s *signal) Witness() string {
	return s.handle
}

