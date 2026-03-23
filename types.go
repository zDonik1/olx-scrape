package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
	Id             uint                    `json:"id"`
	Date           Date                    `json:"date"`
	Price          float32                 `json:"price"`
	Condition      Condition               `json:"condition"`
	Name           string                  `json:"name"`
	Desc           string                  `json:"desc"`
	Url            string                  `json:"url"`
	StructuredData OrderedMap[string, any] `json:"data"`
}

func (ad AdData) CsvHeaders() []string {
	return slices.Concat(
		[]string{"id", "date", "price", "condition", "name", "desc", "url"},
		ad.StructuredData.Keys(),
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

	for _, v := range ad.StructuredData.All() {
		result = append(result, stringify(v))
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

func NewOrderedMap[K comparable, V any]() OrderedMap[K, V] {
	return OrderedMap[K, V]{
		m:     make(map[K]V),
		order: make([]K, 0),
	}
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

func (om OrderedMap[K, V]) MarshalJSON() ([]byte, error) {
	result := make([]string, 0, len(om.Keys()))
	for k, v := range om.All() {
		m, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		result = append(result, fmt.Sprintf(`"%s":%s`, stringify(k), m))
	}
	return fmt.Appendf(nil, "{%s}", strings.Join(result, ",")), nil
}

func (om *OrderedMap[K, V]) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	data, found := bytes.CutPrefix(data, []byte("{"))
	if !found {
		return errors.New("'{' not found")
	}
	data, found = bytes.CutSuffix(data, []byte("}"))
	if !found {
		return errors.New("'}' not found")
	}

	om.m = make(map[K]V)
	om.order = make([]K, 0)

	keyVals := bytes.SplitSeq(data, []byte(","))
	for keyVal := range keyVals {
		keyValPair := bytes.Split(keyVal, []byte(":"))
		if len(keyValPair) != 2 {
			return fmt.Errorf("%s not key-value pair", keyVal)
		}

		key := bytes.TrimSpace(keyValPair[0])
		key, found := bytes.CutPrefix(key, []byte(`"`))
		if !found {
			return errors.New(`'"' not found near key`)
		}
		key, found = bytes.CutSuffix(key, []byte(`"`))
		if !found {
			return errors.New(`'"' not found near key`)
		}

		k, ok := any(string(key)).(K)
		if !ok {
			return fmt.Errorf("key type K is not of type string")
		}

		var v V
		if err := json.Unmarshal(keyValPair[1], &v); err != nil {
			return err
		}

		om.Set(k, v)
	}
	return nil
}
