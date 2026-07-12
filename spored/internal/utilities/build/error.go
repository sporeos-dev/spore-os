package build

import (
	"fmt"
	"spored/internal/utilities/error"
	"strings"
)

func Error(err *error.Error, args ...any) string {
	argsstr := ""
	if len(args) > 0 {
		pairs := make([]string, 0, len(args)/2)
		for i := 0; i+1 < len(args); i += 2 {
			pairs = append(pairs, fmt.Sprintf("%v=\"%v\"", args[i], args[i+1]))
		}
		argsstr = " " + strings.Join(pairs, " ")
	}
	return fmt.Sprintf(`error code=%s what="%s"%s` + "\n", err.Code, err.What, argsstr)
}
