package types

import "golang.org/x/exp/constraints"

type Set[T constraints.Ordered] map[T]bool

// Slice returns slice of strings
func (ss Set[T]) Slice() []T {
	result := make([]T, 0, len(ss))
	for key := range ss {
		result = append(result, key)
	}
	return result
}

// Add adds a value
func (ss *Set[T]) Add(s T) *Set[T] {
	(*ss)[s] = true
	return ss
}

// Remove removes   a value
func (ss *Set[T]) Remove(s T) *Set[T] {
	delete(*ss, s)
	return ss
}
