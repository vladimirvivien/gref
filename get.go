package gref

import (
	"fmt"
	"reflect"
)

// ============================================================================
// Core Extraction
// ============================================================================

// Get extracts a typed value from any Valuable.
// Returns an error if the types don't match or cannot be converted.
//
//	name, err := refl.Get[string](refl.From(user).Struct().Field("Name"))
//	age, err := refl.Get[int](refl.From(user).Struct().Field("Age"))
func Get[T any](v Valuable) (T, error) {
	var zero T
	rv := v.reflectValue()

	if !rv.IsValid() {
		return zero, ErrNilValue
	}

	targetType := reflect.TypeOf(zero)

	// Handle interface{} target type
	if targetType == nil {
		targetType = reflect.TypeOf((*any)(nil)).Elem()
	}

	// Direct type match
	if rv.Type() == targetType {
		return rv.Interface().(T), nil
	}

	// Try assignment
	if rv.Type().AssignableTo(targetType) {
		result := reflect.New(targetType).Elem()
		result.Set(rv)
		return result.Interface().(T), nil
	}

	// Try conversion
	if rv.Type().ConvertibleTo(targetType) {
		converted := rv.Convert(targetType)
		return converted.Interface().(T), nil
	}

	return zero, fmt.Errorf("%w: cannot convert %s to %s", ErrTypeMismatch, rv.Type(), targetType)
}

// MustGet extracts a typed value, panicking on error.
//
//	name := refl.MustGet[string](refl.From(user).Struct().Field("Name"))
func MustGet[T any](v Valuable) T {
	result, err := Get[T](v)
	if err != nil {
		panic(err)
	}
	return result
}

// ============================================================================
// Result Type for Chainable Extraction
// ============================================================================

// Result holds the outcome of a TryGet operation.
// Use its methods to handle success/failure cases.
type Result[T any] struct {
	value T
	ok    bool
}

// TryGet attempts to extract a typed value, returning a Result.
// Use Result methods for chainable access patterns.
//
//	val := refl.TryGet[string](field).Or("default")
//	val := refl.TryGet[int](field).OrZero()
//	if refl.TryGet[string](field).Ok() { ... }
func TryGet[T any](v Valuable) Result[T] {
	result, err := Get[T](v)
	return Result[T]{value: result, ok: err == nil}
}

// Or returns the extracted value if successful, otherwise returns the default.
//
//	name := refl.TryGet[string](field).Or("anonymous")
func (r Result[T]) Or(defaultVal T) T {
	if r.ok {
		return r.value
	}
	return defaultVal
}

// OrZero returns the extracted value if successful, otherwise returns the zero value.
//
//	count := refl.TryGet[int](field).OrZero()
func (r Result[T]) OrZero() T {
	if r.ok {
		return r.value
	}
	var zero T
	return zero
}

// OrElse returns the extracted value if successful, otherwise calls fn for the default.
// Useful for expensive default computations.
//
//	val := refl.TryGet[Config](field).OrElse(loadDefaultConfig)
func (r Result[T]) OrElse(fn func() T) T {
	if r.ok {
		return r.value
	}
	return fn()
}

// Ok returns true if extraction was successful.
//
//	if refl.TryGet[string](field).Ok() { ... }
func (r Result[T]) Ok() bool {
	return r.ok
}

// Value returns (value, ok) for traditional Go-style handling.
//
//	if val, ok := refl.TryGet[string](field).Value(); ok { ... }
func (r Result[T]) Value() (T, bool) {
	return r.value, r.ok
}

// Must returns the value or panics if extraction failed.
func (r Result[T]) Must() T {
	if !r.ok {
		var zero T
		panic(fmt.Sprintf("gref: TryGet failed for type %T", zero))
	}
	return r.value
}

// ============================================================================
// Option Type for Presence/Absence
// ============================================================================

// Option holds a value that may or may not be present.
// Used by Try* methods for graceful handling of missing data.
type Option[T any] struct {
	value T
	some  bool
}

// Some creates an Option containing a value.
func Some[T any](value T) Option[T] {
	return Option[T]{value: value, some: true}
}

// None creates an empty Option.
func None[T any]() Option[T] {
	return Option[T]{some: false}
}

// Some returns true if the Option contains a value.
func (o Option[T]) Some() bool {
	return o.some
}

// None returns true if the Option is empty.
func (o Option[T]) None() bool {
	return !o.some
}

// Or returns the contained value if present, otherwise returns the default.
//
//	field := s.TryField("Name").Or(defaultField)
func (o Option[T]) Or(defaultVal T) T {
	if o.some {
		return o.value
	}
	return defaultVal
}

// OrZero returns the contained value if present, otherwise returns the zero value.
//
//	elem := sl.TryIndex(i).OrZero()
func (o Option[T]) OrZero() T {
	if o.some {
		return o.value
	}
	var zero T
	return zero
}

// OrElse returns the contained value if present, otherwise calls fn for the default.
//
//	entry := m.TryGet("key").OrElse(createDefault)
func (o Option[T]) OrElse(fn func() T) T {
	if o.some {
		return o.value
	}
	return fn()
}

// Value returns (value, ok) for traditional Go-style handling.
//
//	if field, ok := s.TryField("Name").Value(); ok { ... }
func (o Option[T]) Value() (T, bool) {
	return o.value, o.some
}

// Must returns the value or panics if empty.
func (o Option[T]) Must() T {
	if !o.some {
		var zero T
		panic(fmt.Sprintf("gref: Option[%T] is empty", zero))
	}
	return o.value
}

// ============================================================================
// Type Checking
// ============================================================================

// Is checks if the value can be extracted as type T.
// This is a convenient shorthand for checking type compatibility.
//
//	if refl.Is[int](v) {
//	    n := refl.MustGet[int](v)
//	}
func Is[T any](v Valuable) bool {
	_, err := Get[T](v)
	return err == nil
}

// IsExactly checks if the value is exactly the given type (no conversion).
//
//	refl.IsExactly[int64](v)  // true only if v is int64, not int
func IsExactly[T any](v Valuable) bool {
	rv := v.reflectValue()
	if !rv.IsValid() {
		return false
	}
	targetType := reflect.TypeOf((*T)(nil)).Elem()
	return rv.Type() == targetType
}

// IsKind checks if the value has the given kind.
//
//	refl.IsKind(v, reflect.Struct)
func IsKind(v Valuable, k Kind) bool {
	rv := v.reflectValue()
	if !rv.IsValid() {
		return k == Invalid
	}
	return rv.Kind() == k
}
