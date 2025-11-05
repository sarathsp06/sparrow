package types

// ToPtr returns pointer to passed value
func Ptr[T any](v T) *T {
	return &v
}

// PtrIfNotZero returns pointer to passed value if it is not zero
// It does not work with slices and maps
func PtrIfNotZero[T comparable](v T) *T {
	var zero T
	// even if comparable if it implements IsZero() bool
	// it should use that method instead
	if isZeroImplementation, ok := any(v).(interface{ IsZero() bool }); ok {
		if isZeroImplementation.IsZero() {
			return nil
		}
		return &v
	}
	if v == zero {
		return nil
	}

	return &v
}

// Deref returns derefereced value of v
// if v is nil then zero value is returned
func Deref[T any](v *T) T {
	var zero T
	if v == nil {
		return zero
	}
	return *v
}
