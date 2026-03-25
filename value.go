package gref

import (
	"fmt"
	"reflect"
)

// Value wraps a reflect.Value and provides navigation to specialized types.
// Create with From(v).
//
// Value is the entry point for reflection operations. It preserves the original
// kind of the value - use .Struct(), .Slice(), .Map() etc. to narrow to a
// specialized type with kind-specific methods.
//
// Example:
//
//	v := refl.From(myStruct)  // Value
//	s := v.Struct()           // Struct - panics if not a struct
//	f := s.Field("Name")      // Field - has Tag() method
type Value struct {
	rv reflect.Value
}

// From creates a Value from any Go value.
// This is the main entry point to the refl package.
//
// Example:
//
//	v := refl.From(myStruct)
//	s := v.Struct()  // navigate to Struct operations
//
//	v := refl.From(mySlice)
//	sl := v.Slice()  // navigate to Slice operations
func From(v any) Value {
	if v == nil {
		return Value{rv: reflect.Value{}}
	}
	return Value{rv: reflect.ValueOf(v)}
}

// FromValue creates a Value from an existing reflect.Value.
func FromValue(rv reflect.Value) Value {
	return Value{rv: rv}
}

// reflectValue implements Valuable.
func (v Value) reflectValue() reflect.Value {
	return v.rv
}

// --- Type Information ---

// Kind returns the reflect.Kind of the value.
func (v Value) Kind() Kind {
	if !v.rv.IsValid() {
		return Invalid
	}
	return v.rv.Kind()
}

// Type returns the reflect.Type of the value.
func (v Value) Type() reflect.Type {
	if !v.rv.IsValid() {
		return nil
	}
	return v.rv.Type()
}

// IsValid returns true if the value is valid (not a zero Value).
func (v Value) IsValid() bool {
	return v.rv.IsValid()
}

// IsNil returns true if the value is nil.
func (v Value) IsNil() bool {
	if !v.rv.IsValid() {
		return true
	}
	switch v.rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.rv.IsNil()
	default:
		return false
	}
}

// IsZero returns true if the value is the zero value of its type.
func (v Value) IsZero() bool {
	if !v.rv.IsValid() {
		return true
	}
	return v.rv.IsZero()
}

// Interface returns the value as any.
// Panics if the value cannot be retrieved (e.g., unexported field).
func (v Value) Interface() any {
	if !v.rv.IsValid() {
		return nil
	}
	return v.rv.Interface()
}

// Reflect returns the underlying reflect.Value for advanced operations.
func (v Value) Reflect() reflect.Value {
	return v.rv
}

// CanSet returns true if the value can be modified.
func (v Value) CanSet() bool {
	return v.rv.IsValid() && v.rv.CanSet()
}

// --- Navigation to Specialized Types ---

// Struct returns a Struct for struct operations.
// Automatically dereferences pointers to structs.
// Panics if the value is not a struct (or pointer to struct).
//
// Example:
//
//	s := refl.From(&user).Struct()  // auto-derefs pointer
//	name, _ := refl.Get[string](s.Field("Name"))
func (v Value) Struct() Struct {
	rv := deref(v.rv)
	if rv.Kind() != reflect.Struct {
		panic(fmt.Sprintf("gref: expected struct, got %s", rv.Kind()))
	}
	return Struct{rv: rv, rt: rv.Type()}
}

// Slice returns a Slice for slice/array operations.
// Automatically dereferences pointers.
// Panics if the value is not a slice or array.
//
// Example:
//
//	sl := refl.From([]int{1, 2, 3}).Slice()
//	first, _ := refl.Get[int](sl.First())
func (v Value) Slice() Slice {
	rv := deref(v.rv)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		panic(fmt.Sprintf("gref: expected slice or array, got %s", rv.Kind()))
	}
	return Slice{rv: rv, elemType: rv.Type().Elem()}
}

// Map returns a Map for map operations.
// Automatically dereferences pointers.
// Panics if the value is not a map.
//
// Example:
//
//	m := refl.From(map[string]int{"a": 1}).Map()
//	val, _ := refl.Get[int](m.Get("a"))
func (v Value) Map() Map {
	rv := deref(v.rv)
	if rv.Kind() != reflect.Map {
		panic(fmt.Sprintf("gref: expected map, got %s", rv.Kind()))
	}
	return Map{rv: rv, keyType: rv.Type().Key(), valType: rv.Type().Elem()}
}

// Chan returns a Chan for channel operations.
// Automatically dereferences pointers.
// Panics if the value is not a channel.
//
// Example:
//
//	c := refl.From(make(chan int, 3)).Chan()
//	c.Send(42)
func (v Value) Chan() Chan {
	rv := deref(v.rv)
	if rv.Kind() != reflect.Chan {
		panic(fmt.Sprintf("gref: expected chan, got %s", rv.Kind()))
	}
	return Chan{rv: rv, elemType: rv.Type().Elem(), dir: rv.Type().ChanDir()}
}

// Func returns a Func for function operations.
// Automatically dereferences pointers.
// Panics if the value is not a function.
//
// Example:
//
//	f := refl.From(strings.ToUpper).Func()
//	result := f.Call("hello")
func (v Value) Func() Func {
	rv := deref(v.rv)
	if rv.Kind() != reflect.Func {
		panic(fmt.Sprintf("gref: expected func, got %s", rv.Kind()))
	}
	return Func{rv: rv, rt: rv.Type()}
}

// Ptr returns a Ptr for explicit pointer operations.
// Does NOT auto-dereference - use this when you need pointer semantics.
// Panics if the value is not a pointer.
//
// Example:
//
//	p := refl.From(&x).Ptr()
//	p.Set(42)  // sets *x = 42
func (v Value) Ptr() Ptr {
	if v.rv.Kind() != reflect.Pointer {
		panic(fmt.Sprintf("gref: expected pointer, got %s", v.rv.Kind()))
	}
	return Ptr{rv: v.rv, elemType: v.rv.Type().Elem()}
}

// Iface returns an Iface for interface operations.
// Panics if the value is not an interface.
//
// Example:
//
//	var r io.Reader = strings.NewReader("hello")
//	iface := refl.From(&r).Ptr().Elem().Iface()
//	concrete := iface.ConcreteType()
func (v Value) Iface() Iface {
	if v.rv.Kind() != reflect.Interface {
		panic(fmt.Sprintf("gref: expected interface, got %s", v.rv.Kind()))
	}
	return Iface{rv: v.rv}
}

// --- Convenience Methods for Primitives ---
// These panic if the kind doesn't match.

// String returns the value as string. Panics if not a string.
func (v Value) String() string {
	rv := deref(v.rv)
	if rv.Kind() != reflect.String {
		panic(fmt.Sprintf("gref: expected string, got %s", rv.Kind()))
	}
	return rv.String()
}

// Int returns the value as int64. Panics if not an integer type.
func (v Value) Int() int64 {
	rv := deref(v.rv)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int()
	default:
		panic(fmt.Sprintf("gref: expected int, got %s", rv.Kind()))
	}
}

// Uint returns the value as uint64. Panics if not an unsigned integer type.
func (v Value) Uint() uint64 {
	rv := deref(v.rv)
	switch rv.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return rv.Uint()
	default:
		panic(fmt.Sprintf("gref: expected uint, got %s", rv.Kind()))
	}
}

// Float returns the value as float64. Panics if not a float type.
func (v Value) Float() float64 {
	rv := deref(v.rv)
	switch rv.Kind() {
	case reflect.Float32, reflect.Float64:
		return rv.Float()
	default:
		panic(fmt.Sprintf("gref: expected float, got %s", rv.Kind()))
	}
}

// Bool returns the value as bool. Panics if not a bool.
func (v Value) Bool() bool {
	rv := deref(v.rv)
	if rv.Kind() != reflect.Bool {
		panic(fmt.Sprintf("gref: expected bool, got %s", rv.Kind()))
	}
	return rv.Bool()
}

// Bytes returns the value as []byte. Panics if not []byte.
func (v Value) Bytes() []byte {
	rv := deref(v.rv)
	if rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() == reflect.Uint8 {
		return rv.Bytes()
	}
	panic(fmt.Sprintf("gref: expected []byte, got %s", rv.Type()))
}

// Len returns the length for strings, slices, arrays, maps, channels.
// Panics if not applicable.
func (v Value) Len() int {
	return deref(v.rv).Len()
}

// Cap returns the capacity for slices, arrays, channels.
// Panics if not applicable.
func (v Value) Cap() int {
	return deref(v.rv).Cap()
}
