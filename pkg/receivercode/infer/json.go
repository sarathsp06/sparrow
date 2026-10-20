package infer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const noteNull = "null in the sample, type unknown"

// FromJSON infers a type from one JSON sample. Every key present in the
// sample is required; array elements are merged into one element type.
func FromJSON(sample []byte) (*Type, error) {
	v, err := decodeJSON(sample)
	if err != nil {
		return nil, err
	}
	return fromValue(v, 0), nil
}

// decodeJSON decodes exactly one JSON value, keeping numbers as json.Number
// so integers and floats can be told apart.
func decodeJSON(data []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("infer: invalid JSON: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("infer: invalid JSON: trailing data after the value")
	}
	return v, nil
}

func fromValue(v any, depth int) *Type {
	if depth > maxDepth {
		return anyType("nested too deeply to type")
	}
	switch v := v.(type) {
	case nil:
		t := anyType(noteNull)
		t.Nullable = true
		return t
	case bool:
		return &Type{Kind: Bool}
	case string:
		return &Type{Kind: String}
	case json.Number:
		if isInt(v) {
			return &Type{Kind: Int}
		}
		return &Type{Kind: Float}
	case map[string]any:
		if len(v) == 0 {
			return &Type{Kind: Map, Elem: anyType(""), Note: "empty object in the sample"}
		}
		fields := make([]Field, 0, len(v))
		for name, val := range v {
			fields = append(fields, Field{Name: name, Type: fromValue(val, depth+1), Required: true})
		}
		sortFields(fields)
		return &Type{Kind: Object, Fields: fields}
	case []any:
		if len(v) == 0 {
			return &Type{Kind: Array, Elem: anyType("empty array in the sample")}
		}
		var elem *Type
		for _, e := range v {
			elem = merge(elem, fromValue(e, depth+1))
		}
		return &Type{Kind: Array, Elem: elem}
	}
	return anyType("")
}

// isInt reports whether n is a whole number that fits an int64.
func isInt(n json.Number) bool {
	s := string(n)
	if strings.ContainsAny(s, ".eE") {
		return false
	}
	_, err := strconv.ParseInt(s, 10, 64)
	return err == nil
}
