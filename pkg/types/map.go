package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"golang.org/x/exp/constraints"
)

type Map[K constraints.Ordered, V any] map[K]V

// Values returns values from the map
func (m Map[_, V]) Values() []V {
	result := make([]V, 0, len(m))
	for _, v := range m {
		result = append(result, v)
	}
	return result
}

// Keys returns keys of the map
func (m Map[K, _]) Keys() []K {
	result := make([]K, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}

var emptyJSON = []byte(`{}`)

// Value returns m as a json string value
func (m Map[K, V]) Value() (driver.Value, error) {
	if len(m) == 0 {
		return emptyJSON, nil
	}
	v, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("types.Map: Value failed wiht %s", err)
	}
	return v, nil
}

// Scan implements the Scanner interface for Map.
// It scans the value into m.
func (m Map[K, V]) Scan(src any) error {
	if m == nil {
		return errors.New("types.Map: nil map passed to scan")
	}
	var source []byte
	switch t := src.(type) {
	case string:
		source = []byte(t)
	case []byte:
		source = t
	case nil:
		return nil
	default:
		return errors.New("types.Map: not compatible type for types.Map")
	}

	return json.Unmarshal(source, &m)
}
