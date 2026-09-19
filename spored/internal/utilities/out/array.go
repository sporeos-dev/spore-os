package out

import (
	"fmt"
	"strings"
)

type array[T any] struct {
	Key   string
	Value []T
}

func Array[T any](key string, value []T) *array[T] {
	return &array[T]{Key: key, Value: value}
}

func (a *array[T]) String() string {
	return fmt.Sprintf("%s=[%s]", a.Key, renderArraySlice(a.Value))
}

func renderArraySlice[T any](s []T) string {
	parts := make([]string, len(s))
	for i, item := range s {
		parts[i] = renderItem(item)
	}
	return strings.Join(parts, ", ")
}