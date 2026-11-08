package error

import (
	"fmt"
	"spored/internal/utilities/out"
)

func Stringify(code Code, module Module, what string, outs []out.IOut) string {
	if len(outs) == 0 {
		return fmt.Sprintf("error code=%s.%s what=%s", code, module, what)
	} else {
		return fmt.Sprintf("error code=%s.%s what=%s %s", code, module, what, out.Join(outs))
	}
}