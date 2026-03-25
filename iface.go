package gref

import (
	"fmt"
	"reflect"
)

// Iface provides operations on interface values.
// Create via Value.Iface().
//
// Example:
//
//	var r io.Reader = strings.NewReader("hello")
//	iface := refl.From(&r).Ptr().Elem().Iface()
//	concreteType := iface.ConcreteType()  // *strings.Reader
type Iface struct {
	rv reflect.Value
}

// reflectValue implements Valuable.
func (i Iface) reflectValue() reflect.Value {
	return i.rv
}

// --- Nil Checks ---

// IsNil returns true if the interface is nil.
func (i Iface) IsNil() bool {
	return i.rv.IsNil()
}

// HasTypedNil returns true if the interface holds a typed nil.
// (Interface is not nil, but the contained value is nil.)
//
// Example:
//
//	var p *Person = nil
//	var any interface{} = p  // typed nil
//	iface := refl.From(&any).Ptr().Elem().Iface()
//	iface.HasTypedNil()  // true
func (i Iface) HasTypedNil() bool {
	if i.rv.IsNil() {
		return false
	}
	elem := i.rv.Elem()
	switch elem.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer, reflect.Slice:
		return elem.IsNil()
	default:
		return false
	}
}

// --- Type Information ---

// Type returns the interface type.
func (i Iface) Type() reflect.Type {
	return i.rv.Type()
}

// ConcreteType returns the type of the underlying value.
// Returns nil if the interface is nil.
func (i Iface) ConcreteType() reflect.Type {
	if i.rv.IsNil() {
		return nil
	}
	return i.rv.Elem().Type()
}

// ConcreteKind returns the Kind of the underlying value.
// Returns Invalid if the interface is nil.
func (i Iface) ConcreteKind() Kind {
	if i.rv.IsNil() {
		return Invalid
	}
	return i.rv.Elem().Kind()
}

// --- Underlying Value ---

// Underlying returns the concrete value inside the interface.
// Panics if the interface is nil.
//
// Example:
//
//	v := iface.Underlying()
//	s := v.Struct()  // if underlying is a struct
func (i Iface) Underlying() Value {
	if i.rv.IsNil() {
		panic("gref: nil interface has no underlying value")
	}
	return Value{rv: i.rv.Elem()}
}

// TryUnderlying returns the concrete value wrapped in an Option.
// Returns None if the interface is nil.
//
//	val := iface.TryUnderlying().Or(defaultValue)
//	if val, ok := iface.TryUnderlying().Value(); ok { ... }
func (i Iface) TryUnderlying() Option[Value] {
	if i.rv.IsNil() {
		return None[Value]()
	}
	return Some(Value{rv: i.rv.Elem()})
}

// --- Type Assertions ---

// CanAssertTo checks if the underlying value can be assigned to the target type.
func (i Iface) CanAssertTo(targetType reflect.Type) bool {
	if i.rv.IsNil() {
		return false
	}
	return i.rv.Elem().Type().AssignableTo(targetType)
}

// Implements checks if the underlying value implements an interface.
// Pass a nil pointer of the interface type: (*MyInterface)(nil)
//
// Example:
//
//	if iface.Implements((*fmt.Stringer)(nil)) {
//	    // can call String() method
//	}
func (i Iface) Implements(iface any) bool {
	if i.rv.IsNil() {
		return false
	}

	ifaceType := reflect.TypeOf(iface)
	if ifaceType == nil {
		return false
	}
	if ifaceType.Kind() == reflect.Pointer {
		ifaceType = ifaceType.Elem()
	}
	if ifaceType.Kind() != reflect.Interface {
		return false
	}

	concreteType := i.rv.Elem().Type()
	return concreteType.Implements(ifaceType) || reflect.PointerTo(concreteType).Implements(ifaceType)
}

// --- Navigation ---

// Struct returns the underlying value as a Struct.
// Panics if nil or not a struct.
func (i Iface) Struct() Struct {
	if i.rv.IsNil() {
		panic("gref: nil interface")
	}
	rv := deref(i.rv.Elem())
	if rv.Kind() != reflect.Struct {
		panic(fmt.Sprintf("gref: interface holds %s, not struct", rv.Kind()))
	}
	return Struct{rv: rv, rt: rv.Type()}
}

// Slice returns the underlying value as a Slice.
// Panics if nil or not a slice.
func (i Iface) Slice() Slice {
	if i.rv.IsNil() {
		panic("gref: nil interface")
	}
	rv := deref(i.rv.Elem())
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		panic(fmt.Sprintf("gref: interface holds %s, not slice", rv.Kind()))
	}
	return Slice{rv: rv, elemType: rv.Type().Elem()}
}

// Map returns the underlying value as a Map.
// Panics if nil or not a map.
func (i Iface) Map() Map {
	if i.rv.IsNil() {
		panic("gref: nil interface")
	}
	rv := deref(i.rv.Elem())
	if rv.Kind() != reflect.Map {
		panic(fmt.Sprintf("gref: interface holds %s, not map", rv.Kind()))
	}
	return Map{rv: rv, keyType: rv.Type().Key(), valType: rv.Type().Elem()}
}

// Func returns the underlying value as a Func.
// Panics if nil or not a function.
func (i Iface) Func() Func {
	if i.rv.IsNil() {
		panic("gref: nil interface")
	}
	rv := deref(i.rv.Elem())
	if rv.Kind() != reflect.Func {
		panic(fmt.Sprintf("gref: interface holds %s, not func", rv.Kind()))
	}
	return Func{rv: rv, rt: rv.Type()}
}

// Ptr returns the underlying value as a Ptr.
// Panics if nil or not a pointer.
func (i Iface) Ptr() Ptr {
	if i.rv.IsNil() {
		panic("gref: nil interface")
	}
	rv := i.rv.Elem()
	if rv.Kind() != reflect.Pointer {
		panic(fmt.Sprintf("gref: interface holds %s, not pointer", rv.Kind()))
	}
	return Ptr{rv: rv, elemType: rv.Type().Elem()}
}

// Chan returns the underlying value as a Chan.
// Panics if nil or not a channel.
func (i Iface) Chan() Chan {
	if i.rv.IsNil() {
		panic("gref: nil interface")
	}
	rv := deref(i.rv.Elem())
	if rv.Kind() != reflect.Chan {
		panic(fmt.Sprintf("gref: interface holds %s, not chan", rv.Kind()))
	}
	return Chan{rv: rv, elemType: rv.Type().Elem(), dir: rv.Type().ChanDir()}
}

// --- Methods ---

// Method returns a method by name.
// Panics if nil or method not found.
func (i Iface) Method(name string) Func {
	if i.rv.IsNil() {
		panic("gref: nil interface")
	}
	m := i.rv.MethodByName(name)
	if !m.IsValid() {
		panic(fmt.Sprintf("gref: method %q not found", name))
	}
	return Func{rv: m, rt: m.Type()}
}

// TryMethod returns a method by name wrapped in an Option.
// Returns None if not found or interface is nil.
//
//	method := iface.TryMethod("String").Or(defaultMethod)
//	if method, ok := iface.TryMethod("String").Value(); ok { ... }
func (i Iface) TryMethod(name string) Option[Func] {
	if i.rv.IsNil() {
		return None[Func]()
	}
	m := i.rv.MethodByName(name)
	if !m.IsValid() {
		return None[Func]()
	}
	return Some(Func{rv: m, rt: m.Type()})
}

// NumMethods returns the number of methods.
func (i Iface) NumMethods() int {
	return i.rv.NumMethod()
}

// --- Conversion ---

// Interface returns the interface value as any.
func (i Iface) Interface() any {
	return i.rv.Interface()
}

// Value returns this as a Value.
func (i Iface) Value() Value {
	return Value{rv: i.rv}
}
