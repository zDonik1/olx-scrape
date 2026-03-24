package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderedStringMap_MarshalJSON_KeyValue(t *testing.T) {
	om := NewOrderedStringMapWithData(
		map[string]any{
			"key1": "value1",
			"key2": "value2",
		},
		[]string{"key1", "key2"},
	)

	data, err := json.Marshal(om)
	require.NoError(t, err)

	assert.Equal(t, string(data), `{"key1":"value1","key2":"value2"}`)
}

func TestOrderedStringMap_MarshalJSON_NestedObjects(t *testing.T) {
	om := NewOrderedStringMapWithData(
		map[string]any{
			"obj1": NewOrderedStringMapWithData(
				map[string]any{
					"field1": "val1",
					"field2": "val2",
				},
				[]string{"field1", "field2"},
			),
			"obj2": NewOrderedStringMapWithData(
				map[string]any{
					"field1": "val1",
					"field2": "val2",
				},
				[]string{"field1", "field2"},
			),
		},
		[]string{"obj1", "obj2"},
	)

	data, err := json.Marshal(om)
	require.NoError(t, err)

	assert.Equal(t, string(data),
		`{"obj1":{"field1":"val1","field2":"val2"},`+
			`"obj2":{"field1":"val1","field2":"val2"}}`,
	)
}

func TestOrderedStringMap_MarshalJSON_NestedArrays(t *testing.T) {
	om := NewOrderedStringMapWithData(
		map[string]any{
			"arr1": []any{"elem1", "elem2"},
			"arr2": []any{"elem1", "elem2"},
		},
		[]string{"arr1", "arr2"},
	)

	data, err := json.Marshal(om)
	require.NoError(t, err)

	assert.Equal(t, string(data),
		`{"arr1":["elem1","elem2"],`+
			`"arr2":["elem1","elem2"]}`,
	)
}

func TestOrderedStringMap_UnmarshalJSON_KeyValue(t *testing.T) {
	content := `
{
	"key1": "value1",
	"key2": "value2"
}`

	om := NewOrderedStringMap()
	require.NoError(t, json.Unmarshal([]byte(content), om))

	assert.Equal(t, NewOrderedStringMapWithData(
		map[string]any{
			"key1": "value1",
			"key2": "value2",
		},
		[]string{"key1", "key2"},
	), om)
}

func TestOrderedStringMap_UnmarshalJSON_NestedObjects(t *testing.T) {
	content := `
{
	"obj1": {
		"field1": "val1",
		"field2": "val2"
	},
	"obj2": {
		"field1": "val1",
		"field2": "val2"
	}
}`

	om := NewOrderedStringMap()
	require.NoError(t, json.Unmarshal([]byte(content), om))

	assert.Equal(t, NewOrderedStringMapWithData(
		map[string]any{
			"obj1": NewOrderedStringMapWithData(
				map[string]any{
					"field1": "val1",
					"field2": "val2",
				},
				[]string{"field1", "field2"},
			),
			"obj2": NewOrderedStringMapWithData(
				map[string]any{
					"field1": "val1",
					"field2": "val2",
				},
				[]string{"field1", "field2"},
			),
		},
		[]string{"obj1", "obj2"},
	), om)
}

func TestOrderedStringMap_UnmarshalJSON_NestedArrays(t *testing.T) {
	content := `
{
	"arr1": [
		"elem1",
		"elem2"
	],
	"arr2": [
		"elem1",
		"elem2"
	]
}`

	om := NewOrderedStringMap()
	require.NoError(t, json.Unmarshal([]byte(content), om))

	assert.Equal(t, NewOrderedStringMapWithData(
		map[string]any{
			"arr1": []any{"elem1", "elem2"},
			"arr2": []any{"elem1", "elem2"},
		},
		[]string{"arr1", "arr2"},
	), om)
}
