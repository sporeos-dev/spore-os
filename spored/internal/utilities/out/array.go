package out

import (
	"fmt"
	"strings"
)

type Array[T any] struct {
	Key   string
	Value []T
}

func NewArray[T any](key string, value []T) *Array[T] {
	return &Array[T]{Key: key, Value: value}
}

func (a *Array[T]) String() string {
	return fmt.Sprintf("%s=[%s]", a.Key, renderArraySlice(a.Value))
}

func renderArraySlice[T any](s []T) string {
	parts := make([]string, len(s))
	for i, item := range s {
		parts[i] = renderItem(item)
	}
	return strings.Join(parts, " ")
}