package gref

import (
	"fmt"
	"iter"
	"reflect"
)

// Slice provides operations on slice and array values.
// Create via Value.Slice() or MakeSlice[T]().
//
// Example:
//
//	sl := refl.From([]int{1, 2, 3}).Slice()
//	first, _ := refl.Get[int](sl.First())
//
//	// Or create dynamically
//	sl := refl.MakeSlice[string](0, 10)
//	sl = sl.Append("hello")
type Slice struct {
	rv       reflect.Value
	elemType reflect.Type
}

// reflectValue implements Valuable.
func (s Slice) reflectValue() reflect.Value {
	return s.rv
}

// --- Size Information ---

// Len returns the length.
func (s Slice) Len() int {
	return s.rv.Len()
}

// Cap returns the capacity.
func (s Slice) Cap() int {
	return s.rv.Cap()
}

// IsEmpty returns true if length is 0.
func (s Slice) IsEmpty() bool {
	return s.rv.Len() == 0
}

// IsNil returns true if the slice is nil (always false for arrays).
func (s Slice) IsNil() bool {
	if s.rv.Kind() == reflect.Array {
		return false
	}
	return s.rv.IsNil()
}

// IsArray returns true if this is an array (not a slice).
func (s Slice) IsArray() bool {
	return s.rv.Kind() == reflect.Array
}

// --- Type Information ---

// ElemType returns the element type.
func (s Slice) ElemType() reflect.Type {
	return s.elemType
}

// Type returns the slice/array type.
func (s Slice) Type() reflect.Type {
	return s.rv.Type()
}

// --- Element Access ---

// Index returns the element at index i.
// Panics if index is out of bounds.
//
// Example:
//
//	elem := sl.Index(2)
//	val, _ := refl.Get[int](elem)
//	pos := elem.Position()  // 2
func (s Slice) Index(i int) Element {
	if i < 0 || i >= s.rv.Len() {
		panic(fmt.Sprintf("gref: index %d out of bounds [0:%d]", i, s.rv.Len()))
	}
	return Element{rv: s.rv.Index(i), index: i, elemType: s.elemType}
}

// TryIndex returns the element at index i wrapped in an Option.
// Returns None if index is out of bounds.
//
//	elem := sl.TryIndex(i).Or(defaultElem)
//	if elem, ok := sl.TryIndex(i).Value(); ok { ... }
func (s Slice) TryIndex(i int) Option[Element] {
	if i < 0 || i >= s.rv.Len() {
		return None[Element]()
	}
	return Some(Element{rv: s.rv.Index(i), index: i, elemType: s.elemType})
}

// First returns the first element. Panics if empty.
func (s Slice) First() Element {
	return s.Index(0)
}

// Last returns the last element. Panics if empty.
func (s Slice) Last() Element {
	return s.Index(s.rv.Len() - 1)
}

// --- Modification ---

// Set sets the element at index i. Returns the slice for chaining.
// Panics if index out of bounds or not settable.
func (s Slice) Set(i int, value any) Slice {
	if i < 0 || i >= s.rv.Len() {
		panic(fmt.Sprintf("gref: index %d out of bounds [0:%d]", i, s.rv.Len()))
	}
	s.rv.Index(i).Set(reflect.ValueOf(value))
	return s
}

// Append appends values and returns the new slice.
// Panics if this is an array.
//
// Example:
//
//	sl = sl.Append(4, 5, 6)
func (s Slice) Append(values ...any) Slice {
	if s.rv.Kind() == reflect.Array {
		panic("gref: cannot append to array")
	}
	newSlice := s.rv
	for _, v := range values {
		newSlice = reflect.Append(newSlice, reflect.ValueOf(v))
	}
	return Slice{rv: newSlice, elemType: s.elemType}
}

// AppendSlice appends another slice.
func (s Slice) AppendSlice(other Slice) Slice {
	if s.rv.Kind() == reflect.Array {
		panic("gref: cannot append to array")
	}
	newSlice := reflect.AppendSlice(s.rv, other.rv)
	return Slice{rv: newSlice, elemType: s.elemType}
}

// SubSlice returns a sub-slice [low:high].
func (s Slice) SubSlice(low, high int) Slice {
	return Slice{rv: s.rv.Slice(low, high), elemType: s.elemType}
}

// SubSlice3 returns a sub-slice [low:high:max] with explicit capacity.
func (s Slice) SubSlice3(low, high, max int) Slice {
	return Slice{rv: s.rv.Slice3(low, high, max), elemType: s.elemType}
}

// Clear sets all elements to their zero value.
func (s Slice) Clear() Slice {
	zero := reflect.Zero(s.elemType)
	for i := 0; i < s.rv.Len(); i++ {
		s.rv.Index(i).Set(zero)
	}
	return s
}

// --- Iteration ---

// Each calls fn for each element. Return false to stop.
//
// Example:
//
//	sl.Each(func(e refl.Element) bool {
//	    val, _ := refl.Get[int](e)
//	    fmt.Println(e.Position(), val)
//	    return true  // continue
//	})
func (s Slice) Each(fn func(Element) bool) {
	for i := 0; i < s.rv.Len(); i++ {
		elem := Element{rv: s.rv.Index(i), index: i, elemType: s.elemType}
		if !fn(elem) {
			break
		}
	}
}

// Iter returns an iterator for use with range.
//
// Example:
//
//	for elem := range sl.Iter() {
//	    val, _ := refl.Get[int](elem)
//	    fmt.Println(elem.Position(), val)
//	}
func (s Slice) Iter() iter.Seq[Element] {
	return func(yield func(Element) bool) {
		for i := 0; i < s.rv.Len(); i++ {
			elem := Element{rv: s.rv.Index(i), index: i, elemType: s.elemType}
			if !yield(elem) {
				return
			}
		}
	}
}

// Map applies fn to each element and returns a new slice.
//
// Example:
//
//	doubled := sl.Map(func(e refl.Element) any {
//	    val, _ := refl.Get[int](e)
//	    return val * 2
//	})
func (s Slice) Map(fn func(Element) any) Slice {
	if s.rv.Len() == 0 {
		return s
	}

	// Determine result type from first element
	first := fn(Element{rv: s.rv.Index(0), index: 0, elemType: s.elemType})
	firstVal := reflect.ValueOf(first)
	resultType := firstVal.Type()

	result := reflect.MakeSlice(reflect.SliceOf(resultType), s.rv.Len(), s.rv.Len())
	result.Index(0).Set(firstVal)

	for i := 1; i < s.rv.Len(); i++ {
		v := fn(Element{rv: s.rv.Index(i), index: i, elemType: s.elemType})
		result.Index(i).Set(reflect.ValueOf(v))
	}

	return Slice{rv: result, elemType: resultType}
}

// Filter returns a new slice with elements matching the predicate.
//
// Example:
//
//	evens := sl.Filter(func(e refl.Element) bool {
//	    val, _ := refl.Get[int](e)
//	    return val%2 == 0
//	})
func (s Slice) Filter(fn func(Element) bool) Slice {
	result := reflect.MakeSlice(s.rv.Type(), 0, s.rv.Len())

	for i := 0; i < s.rv.Len(); i++ {
		elem := Element{rv: s.rv.Index(i), index: i, elemType: s.elemType}
		if fn(elem) {
			result = reflect.Append(result, s.rv.Index(i))
		}
	}

	return Slice{rv: result, elemType: s.elemType}
}

// Reduce reduces the slice to a single value.
//
// Example:
//
//	sum := sl.Reduce(0, func(acc any, e refl.Element) any {
//	    val, _ := refl.Get[int](e)
//	    return acc.(int) + val
//	}).(int)
func (s Slice) Reduce(initial any, fn func(acc any, elem Element) any) any {
	acc := initial
	for i := 0; i < s.rv.Len(); i++ {
		elem := Element{rv: s.rv.Index(i), index: i, elemType: s.elemType}
		acc = fn(acc, elem)
	}
	return acc
}

// Find returns the first element matching the predicate, or ok=false.
func (s Slice) Find(fn func(Element) bool) (Element, bool) {
	for i := 0; i < s.rv.Len(); i++ {
		elem := Element{rv: s.rv.Index(i), index: i, elemType: s.elemType}
		if fn(elem) {
			return elem, true
		}
	}
	return Element{}, false
}

// FindIndex returns the index of the first matching element, or -1.
func (s Slice) FindIndex(fn func(Element) bool) int {
	for i := 0; i < s.rv.Len(); i++ {
		elem := Element{rv: s.rv.Index(i), index: i, elemType: s.elemType}
		if fn(elem) {
			return i
		}
	}
	return -1
}

// Contains returns true if the slice contains the value.
func (s Slice) Contains(target any) bool {
	tv := reflect.ValueOf(target)
	for i := 0; i < s.rv.Len(); i++ {
		if reflect.DeepEqual(s.rv.Index(i).Interface(), tv.Interface()) {
			return true
		}
	}
	return false
}

// All returns true if all elements match the predicate.
func (s Slice) All(fn func(Element) bool) bool {
	for i := 0; i < s.rv.Len(); i++ {
		elem := Element{rv: s.rv.Index(i), index: i, elemType: s.elemType}
		if !fn(elem) {
			return false
		}
	}
	return true
}

// Any returns true if any element matches the predicate.
func (s Slice) Any(fn func(Element) bool) bool {
	for i := 0; i < s.rv.Len(); i++ {
		elem := Element{rv: s.rv.Index(i), index: i, elemType: s.elemType}
		if fn(elem) {
			return true
		}
	}
	return false
}

// Reverse returns a new slice with elements in reverse order.
func (s Slice) Reverse() Slice {
	n := s.rv.Len()
	result := reflect.MakeSlice(s.rv.Type(), n, n)
	for i := 0; i < n; i++ {
		result.Index(i).Set(s.rv.Index(n - 1 - i))
	}
	return Slice{rv: result, elemType: s.elemType}
}

// --- Conversion ---

// Interface returns the slice as any.
func (s Slice) Interface() any {
	return s.rv.Interface()
}

// Value returns this slice as a Value.
func (s Slice) Value() Value {
	return Value{rv: s.rv}
}

// Ptr returns a pointer to the slice.
// Useful for chaining: refl.MakeSlice[int](0, 10).Append(1,2,3).Ptr()
func (s Slice) Ptr() any {
	ptr := reflect.New(s.rv.Type())
	ptr.Elem().Set(s.rv)
	return ptr.Interface()
}

// Collect returns all elements as []any.
func (s Slice) Collect() []any {
	result := make([]any, s.rv.Len())
	for i := 0; i < s.rv.Len(); i++ {
		result[i] = s.rv.Index(i).Interface()
	}
	return result
}

// ============================================================================
// Element
// ============================================================================

// Element represents a slice/array element with position metadata.
// Created via Slice.Index(), Slice.First(), Slice.Last().
//
// Example:
//
//	elem := sl.Index(2)
//	pos := elem.Position()       // 2
//	val, _ := refl.Get[int](elem) // the value
type Element struct {
	rv       reflect.Value
	index    int
	elemType reflect.Type
}

// reflectValue implements Valuable.
func (e Element) reflectValue() reflect.Value {
	return e.rv
}

// --- Metadata ---

// Position returns the element's index in the slice.
func (e Element) Position() int {
	return e.index
}

// Type returns the element's type.
func (e Element) Type() reflect.Type {
	return e.elemType
}

// --- Value Operations ---

// Kind returns the element's kind.
func (e Element) Kind() Kind {
	return e.rv.Kind()
}

// Interface returns the element value as any.
func (e Element) Interface() any {
	return e.rv.Interface()
}

// IsNil returns true if the element is nil.
func (e Element) IsNil() bool {
	switch e.rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return e.rv.IsNil()
	default:
		return false
	}
}

// IsZero returns true if the element is zero.
func (e Element) IsZero() bool {
	return e.rv.IsZero()
}

// CanSet returns true if the element can be modified.
func (e Element) CanSet() bool {
	return e.rv.CanSet()
}

// Set sets the element value.
func (e Element) Set(value any) {
	if !e.rv.CanSet() {
		panic("gref: element is not settable")
	}
	e.rv.Set(reflect.ValueOf(value))
}

// Value returns the element as a Value.
func (e Element) Value() Value {
	return Value{rv: e.rv}
}

// --- Navigation ---

// Struct returns the element as a Struct.
// Panics if not a struct.
//
// Example:
//
//	// For []User
//	user := sl.Index(0).Struct()
//	name, _ := refl.Get[string](user.Field("Name"))
func (e Element) Struct() Struct {
	rv := deref(e.rv)
	if rv.Kind() != reflect.Struct {
		panic(fmt.Sprintf("gref: element at %d is %s, not struct", e.index, rv.Kind()))
	}
	return Struct{rv: rv, rt: rv.Type()}
}

// Slice returns the element as a Slice.
// Panics if not a slice.
func (e Element) Slice() Slice {
	rv := deref(e.rv)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		panic(fmt.Sprintf("gref: element at %d is %s, not slice", e.index, rv.Kind()))
	}
	return Slice{rv: rv, elemType: rv.Type().Elem()}
}

// Map returns the element as a Map.
// Panics if not a map.
func (e Element) Map() Map {
	rv := deref(e.rv)
	if rv.Kind() != reflect.Map {
		panic(fmt.Sprintf("gref: element at %d is %s, not map", e.index, rv.Kind()))
	}
	return Map{rv: rv, keyType: rv.Type().Key(), valType: rv.Type().Elem()}
}

// Ptr returns the element as a Ptr.
// Panics if not a pointer.
func (e Element) Ptr() Ptr {
	if e.rv.Kind() != reflect.Pointer {
		panic(fmt.Sprintf("gref: element at %d is %s, not pointer", e.index, e.rv.Kind()))
	}
	return Ptr{rv: e.rv, elemType: e.rv.Type().Elem()}
}
