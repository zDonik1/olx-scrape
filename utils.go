package main

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func stringify(v any) string {
	if v == nil {
		return ""
	}

	switch v := v.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case uint:
		return strconv.Itoa(int(v))
	case int:
		return strconv.Itoa(v)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case time.Time:
		return v.Format(time.DateTime)
	}

	if s, ok := v.(fmt.Stringer); ok {
		return s.String()
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.String:
		return rv.String()
	case reflect.Slice, reflect.Array:
		collector := []string{}
		for i := 0; i < rv.Len(); i++ {
			elem := rv.Index(i)
			collector = append(collector, stringify(elem.Interface()))
		}
		return strings.Join(collector, "\n")
	}

	return fmt.Sprintf("%v", v)
}
