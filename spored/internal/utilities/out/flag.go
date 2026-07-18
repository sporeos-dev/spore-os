package out

type Flag struct {
	Flag string
}

func NewFlag(flag string) *Flag {
	return &Flag{
		Flag: flag,
	}
}

func (f *Flag) String() string {
	return f.Flag
}
