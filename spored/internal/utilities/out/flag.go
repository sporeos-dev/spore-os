package out

type _flag struct {
	Flag string
}

func Flag(flag string) *_flag {
	return &_flag{
		Flag: flag,
	}
}

func (f *_flag) String() string {
	return f.Flag
}
