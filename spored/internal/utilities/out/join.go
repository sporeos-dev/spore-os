package out

func Join(outs []IOut) string {
	s := ""
	for i, o := range outs {
		if i > 0 {
			s += " "
		}
		s += o.String()
	}
	return s
}
