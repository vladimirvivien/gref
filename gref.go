// Package gref provides an ergonomic, hierarchical abstraction over Go's reflect package.
//
// # Design Principles
//
//   - Hierarchical API: Navigate with From(v).Struct().Field("Name").Tag("json")
//   - Type-safe extraction: Get[T](value) returns (T, error) on mismatch
//   - Panic on kind mismatch: .Struct() panics if value isn't a struct (programmer error)
//   - Specialized types: Each phase returns its own type (Struct, Field, Slice, Element, etc.)
//   - Auto-dereferencing: .Struct() transparently handles pointers to structs
//
// # Basic Usage
//
//	type User struct {
//	    Name string `json:"name" validate:"required"`
//	    Age  int    `json:"age"`
//	}
//
//	user := &User{Name: "Alice", Age: 30}
//
//	// Navigate and extract
//	name, err := refl.Get[string](refl.From(user).Struct().Field("Name"))
//
//	// Access field metadata
//	tag := refl.From(user).Struct().Field("Name").Tag("validate")  // "required"
//
//	// Nested navigation with dot notation
//	city, _ := refl.Get[string](refl.From(user).Struct().Field("Address.City"))
//
//	// Or explicit chaining
//	city, _ := refl.Get[string](refl.From(user).Struct().Field("Address").Struct().Field("City"))
//
//	// Create dynamic collections
//	m := refl.MakeMap[string, int]()
//	m.Set("count", 42)
//	val, _ := refl.Get[int](m.Get("count"))
package gref

import (
	"errors"
	"reflect"
)

// Common errors returned by the package.
var (
	ErrNilValue        = errors.New("gref: nil value")
	ErrTypeMismatch    = errors.New("gref: type mismatch")
	ErrFieldNotFound   = errors.New("gref: field not found")
	ErrIndexOutOfRange = errors.New("gref: index out of range")
	ErrKeyNotFound     = errors.New("gref: key not found")
	ErrNotSettable     = errors.New("gref: value is not settable")
	ErrChanClosed      = errors.New("gref: channel closed")
	ErrNotConvertible  = errors.New("gref: types not convertible")
)

// Kind represents the specific kind of Go type.
// Re-exported from reflect for convenience.
type Kind = reflect.Kind

// Type represents a Go type.
// Re-exported from reflect for convenience, allowing users to avoid
// importing reflect directly when working with gref's type system.
type Type = reflect.Type

// Re-export reflect.Kind constants.
// Note: Composite type kinds use "Kind" suffix to avoid shadowing type names.
const (
	Invalid       = reflect.Invalid
	Bool          = reflect.Bool
	Int           = reflect.Int
	Int8          = reflect.Int8
	Int16         = reflect.Int16
	Int32         = reflect.Int32
	Int64         = reflect.Int64
	Uint          = reflect.Uint
	Uint8         = reflect.Uint8
	Uint16        = reflect.Uint16
	Uint32        = reflect.Uint32
	Uint64        = reflect.Uint64
	Uintptr       = reflect.Uintptr
	Float32       = reflect.Float32
	Float64       = reflect.Float64
	Complex64     = reflect.Complex64
	Complex128    = reflect.Complex128
	Array         = reflect.Array
	ChanKind      = reflect.Chan
	FuncKind      = reflect.Func
	Interface     = reflect.Interface
	MapKind       = reflect.Map
	Pointer       = reflect.Pointer
	SliceKind     = reflect.Slice
	String        = reflect.String
	StructKind    = reflect.Struct
	UnsafePointer = reflect.UnsafePointer
)

// ChanDir represents channel direction.
type ChanDir = reflect.ChanDir

const (
	RecvDir ChanDir = reflect.RecvDir
	SendDir ChanDir = reflect.SendDir
	BothDir ChanDir = reflect.BothDir
)

// Valuable is implemented by all types that wrap a reflect.Value.
// This allows Get[T] to work with any of them.
type Valuable interface {
	// reflectValue returns the underlying reflect.Value.
	reflectValue() reflect.Value
}

// deref dereferences pointers until a non-pointer value is reached.
// Panics if a nil pointer is encountered.
func deref(rv reflect.Value) reflect.Value {
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			panic("gref: nil pointer dereference")
		}
		rv = rv.Elem()
	}
	return rv
}

// derefType dereferences pointer types until a non-pointer type is reached.
func derefType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}
