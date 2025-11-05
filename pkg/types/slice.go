package types

import (
	"golang.org/x/exp/constraints"
)

// MapFunc  returns a list constructed by appling a the function f
// to all items in slice.
// The name is inspired from default [higher order functions] in functional
// programming.
//
// [higher order functions]: http://learnyouahaskell.com/higher-order-functions#maps-and-filters
func MapSlice[T any, K any](slice []T, f func(v T) K) []K {
	result := make([]K, 0, len(slice))
	for _, v := range slice {
		result = append(result, f(v))
	}
	return result
}

// Reduce applies the function f to the first two elements of the slice,
// then applies f to the result of the previous application and the third
// element, and so on, until all elements have been consumed.
// The name is inspired from default [higher order functions] in functional
// programming.
//
// Eg:
//
//	Reduce([]int{1,2,3,4}, 0, func(acc, v int) int {
//		return acc + v
//	})
//	// returns 10
//
// [higher order functions]: http://learnyouahaskell.com/higher-order-functions#maps-and-filters
func Reduce[S any, K any](list []S, acc K, f func(acc K, v S) K) K {
	result := acc
	for _, v := range list {
		result = f(result, v)
	}
	return result
}

// Filter filters the slice using f.
// The name is inspired from default [higher order functions] in functional
// programming.
//
// [higher order functions]: http://learnyouahaskell.com/higher-order-functions#maps-and-filters
func Filter[T any](slice []T, f func(v T) bool) []T {
	var result []T
	for _, v := range slice {
		if f(v) {
			result = append(result, v)
		}
	}
	return result
}

// FilterInPlace
// The name is inspired from default [higher order functions] in functional
// programming.
//
// [higher order functions]: http://learnyouahaskell.com/higher-order-functions#maps-and-filters
func FilterInPLace[T any](slice []T, f func(v T) bool) []T {
	var j int
	for _, v := range slice {
		if f(v) {
			slice[j] = v
			j++
		}
	}
	return slice[:j]
}

// Sum return sum of the values in the given list.
// It assumes that the type used in slice can hold sum of it,
// The maximum and minimum values for each type is defined [here]
//
// [here]: https://go.dev/ref/spec#Numeric_types
func Sum[T constraints.Ordered](vals []T) T {
	var zero T
	return Reduce(vals, zero, func(acc T, v T) T { return acc + v })
}

// SliceClone makes a clone of the given slice without
// changing or sharing the original slice
func SliceClone[T any](slice []T) []T {
	clone := make([]T, len(slice))
	copy(clone, slice)
	return clone
}

// PointerSlice converts a slice of any type to slice
// of pointers of the same type
func PointerSlice[T any](slice []T) []*T {
	result := make([]*T, 0, len(slice))
	for _, v := range slice {
		v := v
		result = append(result, &v)
	}
	return result
}

// StringSlice generates a new slice of type V from the input slice s of type []T.
// It is useful when converting a slice of one type to another where the types are string or
// dervied from string
func StringSlice[V, T ~string](s []T) []V {
	if len(s) == 0 {
		return []V{}
	}
	result := make([]V, 0, len(s))
	for _, v := range s {
		result = append(result, V(v))
	}
	return result
}
