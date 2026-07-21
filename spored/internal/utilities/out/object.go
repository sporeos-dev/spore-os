package out

import (
	"fmt"
	"sort"
	"strings"
)

type object struct {
	Key   string
	Value map[string]any
}

func Object(key string, value map[string]any) *object {
	return &object{Key: key, Value: value}
}

func (o *object) String() string {
	return fmt.Sprintf("%s={%s}", o.Key, renderMap(o.Value))
}

func renderMap(m map[string]any) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, renderEntry(k, m[k]))
	}
	return strings.Join(parts, " ")
}

func renderEntry(key string, val any) string {
	switch v := val.(type) {
	case nil:
		return key
	case string:
		return fmt.Sprintf(`%s="%s"`, key, v)
	case map[string]any:
		return fmt.Sprintf("%s={%s}", key, renderMap(v))
	case []any:
		return fmt.Sprintf("%s=[%s]", key, renderSlice(v))
	default:
		return fmt.Sprintf(`%s="%v"`, key, v)
	}
}

func renderSlice(s []any) string {
	parts := make([]string, len(s))
	for i, item := range s {
		parts[i] = renderItem(item)
	}
	return strings.Join(parts, " ")
}

func renderItem(val any) string {
	switch v := val.(type) {
	case string:
		return fmt.Sprintf(`"%s"`, v)
	case map[string]any:
		return fmt.Sprintf("{%s}", renderMap(v))
	case []any:
		return fmt.Sprintf("[%s]", renderSlice(v))
	default:
		return fmt.Sprintf(`"%v"`, v)
	}
}

