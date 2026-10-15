package rest

import (
	"reflect"
	"sync"

	"github.com/danielgtaylor/huma/v2"
)

// Arrays in the API are never null: the spec types them as plain arrays
// (huma.DefaultArrayNullable is off), so a request sending null is rejected
// and a response must send [] instead. Handlers often build output from nil
// slices, so emptySlices rewrites every nil slice in a response body to an
// empty one rather than relying on each handler to remember.
func init() {
	huma.DefaultArrayNullable = false
}

// emptySlices is a huma transformer that returns v with every nil slice,
// at any depth, replaced by an empty slice of the same type.
func emptySlices(_ huma.Context, _ string, v any) (any, error) {
	if v == nil {
		return v, nil
	}
	rv := reflect.ValueOf(v)
	if !typeHasSlices(rv.Type()) {
		return v, nil
	}
	return fillSlices(rv).Interface(), nil
}

var slicesByType sync.Map // reflect.Type -> bool

// typeHasSlices caches hasSlices per type.
func typeHasSlices(t reflect.Type) bool {
	if v, ok := slicesByType.Load(t); ok {
		return v.(bool)
	}
	has := hasSlices(t, map[reflect.Type]bool{})
	slicesByType.Store(t, has)
	return has
}

// hasSlices reports whether values of t can contain a slice, so values
// without any skip the copy. Interface values (event payloads and other
// decoded JSON) are left alone: JSON decoding never produces a nil slice.
func hasSlices(t reflect.Type, seen map[reflect.Type]bool) bool {
	if seen[t] {
		return false
	}
	seen[t] = true
	switch t.Kind() {
	case reflect.Slice:
		return t.Elem().Kind() != reflect.Uint8 // []byte marshals as a string
	case reflect.Pointer, reflect.Array:
		return hasSlices(t.Elem(), seen)
	case reflect.Map:
		return hasSlices(t.Elem(), seen)
	case reflect.Struct:
		for i := range t.NumField() {
			if t.Field(i).IsExported() && hasSlices(t.Field(i).Type, seen) {
				return true
			}
		}
	}
	return false
}

// fillSlices returns a copy of v with nil slices replaced by empty ones.
// It copies rather than mutates, so handler-owned values stay untouched.
func fillSlices(v reflect.Value) reflect.Value {
	if !typeHasSlices(v.Type()) {
		return v
	}
	switch v.Kind() {
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return v
		}
		if v.IsNil() {
			return reflect.MakeSlice(v.Type(), 0, 0)
		}
		out := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		for i := range v.Len() {
			out.Index(i).Set(fillSlices(v.Index(i)))
		}
		return out
	case reflect.Pointer:
		if v.IsNil() {
			return v
		}
		out := reflect.New(v.Type().Elem())
		out.Elem().Set(fillSlices(v.Elem()))
		return out
	case reflect.Struct:
		out := reflect.New(v.Type()).Elem()
		out.Set(v)
		for i := range v.NumField() {
			if !v.Type().Field(i).IsExported() {
				continue
			}
			out.Field(i).Set(fillSlices(v.Field(i)))
		}
		return out
	case reflect.Map:
		if v.IsNil() {
			return v
		}
		out := reflect.MakeMapWithSize(v.Type(), v.Len())
		iter := v.MapRange()
		for iter.Next() {
			out.SetMapIndex(iter.Key(), fillSlices(iter.Value()))
		}
		return out
	}
	return v
}
