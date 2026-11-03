// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package out

import (
	"fmt"
	"strings"
)

type pair struct {
	Key   string
	Value string
}

func Pair(key string, value string) *pair {
	return &pair{
		Key:   key,
		Value: value,
	}
}

func (p *pair) String() string {
	if strings.Contains(p.Value, "\n") {
		return fmt.Sprintf("%s=<<\n%s\n>>", p.Key, p.Value)
	}

	value := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		"\r", `\r`,
		"\t", `\t`,
	).Replace(p.Value)
	return fmt.Sprintf(`%s="%s"`, p.Key, value)
}
