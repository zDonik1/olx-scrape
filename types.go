package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"slices"
	"strings"
	"time"
)

type Condition string

const (
	ConditionUnknown Condition = "unknown"
	ConditionNew     Condition = "new"
	ConditionUsed    Condition = "used"
)

type AdData struct {
	Id             uint              `json:"id"`
	Date           Date              `json:"date"`
	Price          float32           `json:"price"`
	Condition      Condition         `json:"condition"`
	Name           string            `json:"name"`
	Desc           string            `json:"desc"`
	Url            string            `json:"url"`
	StructuredData *OrderedStringMap `json:"data"`
}

func (ad AdData) CsvHeaders() []string {
	structDataKeys := []string{}
	if ad.StructuredData != nil {
		structDataKeys = ad.StructuredData.Keys()
	}
	return slices.Concat(
		[]string{"id", "date", "price", "condition", "name", "desc", "url"}, structDataKeys,
	)
}

func (ad AdData) CsvRow() []string {
	result := []string{
		stringify(ad.Id),
		stringify(ad.Date),
		stringify(ad.Price),
		stringify(ad.Condition),
		ad.Name,
		ad.Desc,
		ad.Url,
	}

	if ad.StructuredData != nil {
		for _, v := range ad.StructuredData.All() {
			result = append(result, stringify(v))
		}
	}
	return result
}

type Date struct {
	time.Time
}

func (d Date) String() string {
	return d.Time.Format(time.DateOnly)
}

type Storage []string

func (s Storage) MarshalCSV() (string, error) {
	return strings.Join([]string(s), "\n"), nil
}

type OrderedMap[K comparable, V any] struct {
	m     map[K]V
	order []K
}

func NewOrderedMap[K comparable, V any]() *OrderedMap[K, V] {
	return NewOrderedMapWithData(make(map[K]V), make([]K, 0))
}

func NewOrderedMapWithData[K comparable, V any](m map[K]V, order []K) *OrderedMap[K, V] {
	return &OrderedMap[K, V]{m: m, order: order}
}

func (om OrderedMap[K, V]) Len() int {
	return len(om.order)
}

func (om *OrderedMap[K, V]) Set(k K, v V) {
	if _, exists := om.m[k]; !exists {
		om.order = append(om.order, k)
	}
	om.m[k] = v
}

func (om OrderedMap[K, V]) Keys() []K {
	return om.order
}

func (om OrderedMap[K, V]) Values() []V {
	result := make([]V, 0, len(om.order))
	for _, v := range om.All() {
		result = append(result, v)
	}
	return result
}

func (om OrderedMap[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for _, k := range om.order {
			if !yield(k, om.m[k]) {
				return
			}
		}
	}
}

type OrderedStringMap struct {
	OrderedMap[string, any]
}

func NewOrderedStringMap() *OrderedStringMap {
	return &OrderedStringMap{
		OrderedMap: *NewOrderedMap[string, any](),
	}
}

func NewOrderedStringMapWithData(m map[string]any, order []string) *OrderedStringMap {
	return &OrderedStringMap{OrderedMap: *NewOrderedMapWithData(m, order)}
}

func (om OrderedStringMap) MarshalJSON() ([]byte, error) {
	result := make([]string, 0, len(om.Keys()))
	for k, v := range om.All() {
		m, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		result = append(result, fmt.Sprintf(`"%s":%s`, k, m))
	}
	return fmt.Appendf(nil, "{%s}", strings.Join(result, ",")), nil
}

func (om *OrderedStringMap) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	if !dec.More() {
		return errors.New("input data empty")
	}

	if om.m == nil {
		om.m = make(map[string]any)
	}
	if om.order == nil {
		om.order = make([]string, 0)
	}

	t, err := dec.Token()
	if err != nil {
		return err
	}

	delim, ok := t.(json.Delim)
	if !ok {
		return fmt.Errorf("first token '%s' not delimeter", t)
	}
	if delim != '{' {
		return fmt.Errorf("first token '%s' not '{'", t)
	}

	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return err
		}

		key, ok := t.(string)
		if !ok {
			return fmt.Errorf("key not found in %v", t)
		}

		reader := dec.Buffered()
		c, err := findNonWhiteSpace(reader)
		if err != nil {
			return err
		}
		if c != ':' {
			return fmt.Errorf("expected ':', got '%b'", c)
		}

		c, err = findNonWhiteSpace(reader)
		if err != nil {
			return err
		}

		var result any
		if c == '{' {
			newOm := NewOrderedStringMap()
			dec.Decode(newOm)
			result = newOm
		} else {
			if err := dec.Decode(&result); err != nil {
				return err
			}
		}
		om.Set(key, result)
	}
	return nil
}

// ---- from Decoder source code (modified) ----

func findNonWhiteSpace(reader io.Reader) (byte, error) {
	c := make([]byte, 1)
	for {
		_, err := reader.Read(c)
		if err != nil {
			return 0, err
		}

		if isSpace(c[0]) {
			continue
		}
		return c[0], nil
	}
}

func isSpace(c byte) bool {
	return c <= ' ' && (c == ' ' || c == '\t' || c == '\r' || c == '\n')
}
