package gref

import (
	"fmt"
	"reflect"
)

// Ptr provides explicit pointer operations.
// Unlike other types, Ptr does NOT auto-dereference.
// Create via Value.Ptr().
//
// Example:
//
//	x := 42
//	p := refl.From(&x).Ptr()
//	p.Set(100)  // x is now 100
//
//	val, _ := refl.Get[int](p.Elem())  // 100
type Ptr struct {
	rv       reflect.Value
	elemType reflect.Type
}

// reflectValue implements Valuable.
func (p Ptr) reflectValue() reflect.Value {
	return p.rv
}

// --- Type Information ---

// ElemType returns the pointed-to type.
func (p Ptr) ElemType() reflect.Type {
	return p.elemType
}

// Type returns the pointer type.
func (p Ptr) Type() reflect.Type {
	return p.rv.Type()
}

// IsNil returns true if the pointer is nil.
func (p Ptr) IsNil() bool {
	return p.rv.IsNil()
}

// --- Indirection ---

// IndirectionDepth returns how many levels of pointer.
// **int returns 2.
func (p Ptr) IndirectionDepth() int {
	depth := 0
	t := p.rv.Type()
	for t.Kind() == reflect.Pointer {
		depth++
		t = t.Elem()
	}
	return depth
}

// UltimateType returns the final non-pointer type.
func (p Ptr) UltimateType() reflect.Type {
	t := p.rv.Type()
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

// --- Dereference ---

// Elem returns the pointed-to value.
// Panics if nil.
//
// Example:
//
//	val, _ := refl.Get[int](p.Elem())
func (p Ptr) Elem() Value {
	if p.rv.IsNil() {
		panic("gref: nil pointer dereference")
	}
	return Value{rv: p.rv.Elem()}
}

// TryElem returns the pointed-to value wrapped in an Option.
// Returns None if the pointer is nil.
//
//	val := p.TryElem().Or(defaultValue)
//	if val, ok := p.TryElem().Value(); ok { ... }
func (p Ptr) TryElem() Option[Value] {
	if p.rv.IsNil() {
		return None[Value]()
	}
	return Some(Value{rv: p.rv.Elem()})
}

// DerefAll dereferences all pointer levels.
// Panics if any pointer in the chain is nil.
//
// Example:
//
//	// For **int
//	val, _ := refl.Get[int](p.DerefAll())
func (p Ptr) DerefAll() Value {
	rv := p.rv
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			panic("gref: nil pointer in chain")
		}
		rv = rv.Elem()
	}
	return Value{rv: rv}
}

// DerefN dereferences exactly n levels.
// Panics if not enough levels or nil encountered.
func (p Ptr) DerefN(n int) Value {
	rv := p.rv
	for i := 0; i < n; i++ {
		if rv.Kind() != reflect.Pointer {
			panic(fmt.Sprintf("gref: expected pointer at level %d, got %s", i, rv.Kind()))
		}
		if rv.IsNil() {
			panic(fmt.Sprintf("gref: nil pointer at level %d", i))
		}
		rv = rv.Elem()
	}
	return Value{rv: rv}
}

// --- Set Operations ---

// Set sets the pointed-to value.
// Panics if nil or not settable.
//
// Example:
//
//	x := 42
//	p := refl.From(&x).Ptr()
//	p.Set(100)  // x is now 100
func (p Ptr) Set(value any) {
	if p.rv.IsNil() {
		panic("gref: cannot set through nil pointer")
	}
	elem := p.rv.Elem()
	if !elem.CanSet() {
		panic("gref: pointed-to value is not settable")
	}
	elem.Set(reflect.ValueOf(value))
}

// SetNil sets the pointer to nil.
// The pointer itself must be addressable.
func (p Ptr) SetNil() {
	if !p.rv.CanSet() {
		panic("gref: pointer is not settable")
	}
	p.rv.Set(reflect.Zero(p.rv.Type()))
}

// --- Allocation ---

// Alloc allocates a new value and sets the pointer to it.
// The pointer must be settable.
func (p Ptr) Alloc() Ptr {
	if !p.rv.CanSet() {
		panic("gref: pointer is not settable")
	}
	newVal := reflect.New(p.elemType)
	p.rv.Set(newVal)
	return p
}

// AllocIfNil allocates only if the pointer is nil.
func (p Ptr) AllocIfNil() Ptr {
	if p.rv.IsNil() {
		return p.Alloc()
	}
	return p
}

// --- Navigation ---

// Struct returns the pointed-to value as a Struct.
// Panics if nil or not a struct.
//
// Example:
//
//	s := refl.From(&user).Ptr().Struct()
func (p Ptr) Struct() Struct {
	if p.rv.IsNil() {
		panic("gref: nil pointer")
	}
	rv := p.rv.Elem()
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			panic("gref: nil pointer in chain")
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		panic(fmt.Sprintf("gref: pointed-to value is %s, not struct", rv.Kind()))
	}
	return Struct{rv: rv, rt: rv.Type()}
}

// Slice returns the pointed-to value as a Slice.
// Panics if nil or not a slice.
func (p Ptr) Slice() Slice {
	if p.rv.IsNil() {
		panic("gref: nil pointer")
	}
	rv := p.rv.Elem()
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			panic("gref: nil pointer in chain")
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		panic(fmt.Sprintf("gref: pointed-to value is %s, not slice", rv.Kind()))
	}
	return Slice{rv: rv, elemType: rv.Type().Elem()}
}

// Map returns the pointed-to value as a Map.
// Panics if nil or not a map.
func (p Ptr) Map() Map {
	if p.rv.IsNil() {
		panic("gref: nil pointer")
	}
	rv := p.rv.Elem()
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			panic("gref: nil pointer in chain")
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Map {
		panic(fmt.Sprintf("gref: pointed-to value is %s, not map", rv.Kind()))
	}
	return Map{rv: rv, keyType: rv.Type().Key(), valType: rv.Type().Elem()}
}

// --- Comparison ---

// Equals checks if two pointers point to the same address.
func (p Ptr) Equals(other Ptr) bool {
	return p.rv.Pointer() == other.rv.Pointer()
}

// --- Conversion ---

// Interface returns the pointer as any.
func (p Ptr) Interface() any {
	return p.rv.Interface()
}

// Value returns this pointer as a Value.
func (p Ptr) Value() Value {
	return Value{rv: p.rv}
}

// UnsafePointer returns the raw pointer address.
func (p Ptr) UnsafePointer() uintptr {
	return p.rv.Pointer()
}
