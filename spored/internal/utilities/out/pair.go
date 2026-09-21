// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package out

import "fmt"

type pair struct {
	Key string
	Value string
}

func Pair(key string, value string) *pair {
	return &pair {
		Key: key,
		Value: value,
	}
}

func (p *pair) String() string {
	return fmt.Sprintf(`%s="%s"`, p.Key, p.Value)
}
