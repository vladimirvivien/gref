package gref

import (
	"fmt"
	"iter"
	"reflect"
	"strings"
)

// Map provides operations on map values.
// Create via Value.Map() or MakeMap[K,V]().
//
// Example:
//
//	m := refl.From(map[string]int{"a": 1}).Map()
//	val, _ := refl.Get[int](m.Get("a"))
//
//	// Or create dynamically
//	m := refl.MakeMap[string, int]()
//	m.Set("count", 42)
type Map struct {
	rv      reflect.Value
	keyType reflect.Type
	valType reflect.Type
}

// reflectValue implements Valuable.
func (m Map) reflectValue() reflect.Value {
	return m.rv
}

// --- Size Information ---

// Len returns the number of entries.
func (m Map) Len() int {
	return m.rv.Len()
}

// IsEmpty returns true if the map has no entries.
func (m Map) IsEmpty() bool {
	return m.rv.Len() == 0
}

// IsNil returns true if the map is nil.
func (m Map) IsNil() bool {
	return m.rv.IsNil()
}

// --- Type Information ---

// KeyType returns the key type.
func (m Map) KeyType() reflect.Type {
	return m.keyType
}

// ValType returns the value type.
func (m Map) ValType() reflect.Type {
	return m.valType
}

// Type returns the map type.
func (m Map) Type() reflect.Type {
	return m.rv.Type()
}

// --- Entry Access ---

// Get returns the entry for a key.
// Panics if key not found.
//
// Example:
//
//	entry := m.Get("a")
//	key, _ := refl.Get[string](entry.Key())  // "a"
//	val, _ := refl.Get[int](entry)           // 1
func (m Map) Get(key any) Entry {
	kv := reflect.ValueOf(key)
	vv := m.rv.MapIndex(kv)
	if !vv.IsValid() {
		panic(fmt.Sprintf("gref: key %v not found in map", key))
	}
	return Entry{key: kv, val: vv, keyType: m.keyType, valType: m.valType}
}

// TryGet returns the entry for a key, or ok=false if not found.
//
// Example:
//
//	entry := m.TryGet("key").Or(defaultEntry)
//	if entry, ok := m.TryGet("key").Value(); ok { ... }
func (m Map) TryGet(key any) Option[Entry] {
	kv := reflect.ValueOf(key)
	vv := m.rv.MapIndex(kv)
	if !vv.IsValid() {
		return None[Entry]()
	}
	return Some(Entry{key: kv, val: vv, keyType: m.keyType, valType: m.valType})
}

// Has returns true if the key exists.
func (m Map) Has(key any) bool {
	kv := reflect.ValueOf(key)
	return m.rv.MapIndex(kv).IsValid()
}

// --- Modification ---

// Set sets a key-value pair. Returns the map for chaining.
//
// Example:
//
//	m.Set("a", 1).Set("b", 2).Set("c", 3)
func (m Map) Set(key, value any) Map {
	kv := reflect.ValueOf(key)
	vv := reflect.ValueOf(value)
	m.rv.SetMapIndex(kv, vv)
	return m
}

// Delete removes a key. Returns the map for chaining.
func (m Map) Delete(key any) Map {
	kv := reflect.ValueOf(key)
	m.rv.SetMapIndex(kv, reflect.Value{})
	return m
}

// Clear removes all entries.
func (m Map) Clear() Map {
	mi := m.rv.MapRange()
	var keys []reflect.Value
	for mi.Next() {
		keys = append(keys, mi.Key())
	}
	for _, k := range keys {
		m.rv.SetMapIndex(k, reflect.Value{})
	}
	return m
}

// --- Keys and Values ---

// Keys returns all keys as a Slice.
func (m Map) Keys() Slice {
	keys := reflect.MakeSlice(reflect.SliceOf(m.keyType), m.rv.Len(), m.rv.Len())
	mi := m.rv.MapRange()
	i := 0
	for mi.Next() {
		keys.Index(i).Set(mi.Key())
		i++
	}
	return Slice{rv: keys, elemType: m.keyType}
}

// Values returns all values as a Slice.
func (m Map) Values() Slice {
	values := reflect.MakeSlice(reflect.SliceOf(m.valType), m.rv.Len(), m.rv.Len())
	mi := m.rv.MapRange()
	i := 0
	for mi.Next() {
		values.Index(i).Set(mi.Value())
		i++
	}
	return Slice{rv: values, elemType: m.valType}
}

// Entries returns all entries as a slice.
func (m Map) Entries() []Entry {
	entries := make([]Entry, 0, m.rv.Len())
	mi := m.rv.MapRange()
	for mi.Next() {
		entries = append(entries, Entry{
			key:     mi.Key(),
			val:     mi.Value(),
			keyType: m.keyType,
			valType: m.valType,
		})
	}
	return entries
}

// --- Iteration ---

// Each calls fn for each entry. Return false to stop.
//
// Example:
//
//	m.Each(func(e refl.Entry) bool {
//	    key, _ := refl.Get[string](e.Key())
//	    val, _ := refl.Get[int](e)
//	    fmt.Println(key, val)
//	    return true  // continue
//	})
func (m Map) Each(fn func(Entry) bool) {
	mi := m.rv.MapRange()
	for mi.Next() {
		entry := Entry{key: mi.Key(), val: mi.Value(), keyType: m.keyType, valType: m.valType}
		if !fn(entry) {
			break
		}
	}
}

// Iter returns an iterator for use with range.
//
// Example:
//
//	for entry := range m.Iter() {
//	    key, _ := refl.Get[string](entry.Key())
//	    val, _ := refl.Get[int](entry)
//	    fmt.Println(key, val)
//	}
func (m Map) Iter() iter.Seq[Entry] {
	return func(yield func(Entry) bool) {
		mi := m.rv.MapRange()
		for mi.Next() {
			entry := Entry{key: mi.Key(), val: mi.Value(), keyType: m.keyType, valType: m.valType}
			if !yield(entry) {
				return
			}
		}
	}
}

// Filter returns a new map with entries matching the predicate.
//
// Example:
//
//	positive := m.Filter(func(e refl.Entry) bool {
//	    val, _ := refl.Get[int](e)
//	    return val > 0
//	})
func (m Map) Filter(fn func(Entry) bool) Map {
	result := reflect.MakeMap(m.rv.Type())
	mi := m.rv.MapRange()
	for mi.Next() {
		entry := Entry{key: mi.Key(), val: mi.Value(), keyType: m.keyType, valType: m.valType}
		if fn(entry) {
			result.SetMapIndex(mi.Key(), mi.Value())
		}
	}
	return Map{rv: result, keyType: m.keyType, valType: m.valType}
}

// MapValues applies fn to each value and returns a new map.
func (m Map) MapValues(fn func(Entry) any) Map {
	if m.rv.Len() == 0 {
		return m
	}

	// Determine result type from first entry
	mi := m.rv.MapRange()
	mi.Next()
	firstEntry := Entry{key: mi.Key(), val: mi.Value(), keyType: m.keyType, valType: m.valType}
	first := fn(firstEntry)
	firstVal := reflect.ValueOf(first)
	newValType := firstVal.Type()

	result := reflect.MakeMap(reflect.MapOf(m.keyType, newValType))
	result.SetMapIndex(mi.Key(), firstVal)

	for mi.Next() {
		entry := Entry{key: mi.Key(), val: mi.Value(), keyType: m.keyType, valType: m.valType}
		v := fn(entry)
		result.SetMapIndex(mi.Key(), reflect.ValueOf(v))
	}

	return Map{rv: result, keyType: m.keyType, valType: newValType}
}

// --- Bulk Operations ---

// Merge returns a new map with entries from both maps.
// Values from other take precedence on collision.
func (m Map) Merge(other Map) Map {
	result := reflect.MakeMap(m.rv.Type())

	// Copy from m
	mi := m.rv.MapRange()
	for mi.Next() {
		result.SetMapIndex(mi.Key(), mi.Value())
	}

	// Copy from other (overwrites)
	mi = other.rv.MapRange()
	for mi.Next() {
		result.SetMapIndex(mi.Key(), mi.Value())
	}

	return Map{rv: result, keyType: m.keyType, valType: m.valType}
}

// Clone returns a shallow copy of the map.
func (m Map) Clone() Map {
	result := reflect.MakeMapWithSize(m.rv.Type(), m.rv.Len())
	mi := m.rv.MapRange()
	for mi.Next() {
		result.SetMapIndex(mi.Key(), mi.Value())
	}
	return Map{rv: result, keyType: m.keyType, valType: m.valType}
}

// --- Conversion ---

// Interface returns the map as any.
func (m Map) Interface() any {
	return m.rv.Interface()
}

// Value returns this map as a Value.
func (m Map) Value() Value {
	return Value{rv: m.rv}
}

// ToGoMap converts to map[string]any.
// Panics if key type is not string.
func (m Map) ToGoMap() map[string]any {
	if m.keyType.Kind() != reflect.String {
		panic(fmt.Sprintf("gref: cannot convert map[%s]... to map[string]any", m.keyType))
	}

	result := make(map[string]any, m.rv.Len())
	mi := m.rv.MapRange()
	for mi.Next() {
		result[mi.Key().String()] = mi.Value().Interface()
	}
	return result
}

// ToStruct populates a struct from the map, matching keys to field names.
// Panics if key type is not string or structPtr is not a pointer to struct.
//
// Example:
//
//	m := gref.From(map[string]any{"Name": "Alice", "Age": 30}).Map()
//	var user User
//	m.ToStruct(&user)
func (m Map) ToStruct(structPtr any) {
	if m.keyType.Kind() != reflect.String {
		panic(fmt.Sprintf("gref: cannot convert map[%s]... to struct", m.keyType))
	}

	rv := reflect.ValueOf(structPtr)
	if rv.Kind() != reflect.Pointer || rv.Elem().Kind() != reflect.Struct {
		panic("gref: ToStruct requires pointer to struct")
	}

	sv := rv.Elem()
	st := sv.Type()

	// Build field lookup: field name -> field index
	fieldMap := make(map[string]int, st.NumField())
	for i := 0; i < st.NumField(); i++ {
		sf := st.Field(i)
		if sf.IsExported() {
			fieldMap[sf.Name] = i
		}
	}

	// Set fields from map
	m.setFieldsFromMap(sv, fieldMap)
}

// ToStructTag populates a struct from the map, matching keys to tag values.
// Fields tagged with "-" are skipped.
// Panics if key type is not string or structPtr is not a pointer to struct.
//
// Example:
//
//	m := gref.From(map[string]any{"name": "Alice", "age": 30}).Map()
//	var user User
//	m.ToStructTag(&user, "json")
func (m Map) ToStructTag(structPtr any, tagName string) {
	if m.keyType.Kind() != reflect.String {
		panic(fmt.Sprintf("gref: cannot convert map[%s]... to struct", m.keyType))
	}

	rv := reflect.ValueOf(structPtr)
	if rv.Kind() != reflect.Pointer || rv.Elem().Kind() != reflect.Struct {
		panic("gref: ToStructTag requires pointer to struct")
	}

	sv := rv.Elem()
	st := sv.Type()

	// Build field lookup: tag value -> field index
	fieldMap := make(map[string]int, st.NumField())
	for i := 0; i < st.NumField(); i++ {
		sf := st.Field(i)
		if !sf.IsExported() {
			continue
		}

		key := sf.Name
		if tagVal := sf.Tag.Get(tagName); tagVal != "" {
			if tagVal == "-" {
				continue
			}
			// Handle "name,options" format
			if idx := strings.Index(tagVal, ","); idx != -1 {
				key = tagVal[:idx]
			} else {
				key = tagVal
			}
		}
		fieldMap[key] = i
	}

	// Set fields from map
	m.setFieldsFromMap(sv, fieldMap)
}

// setFieldsFromMap is a helper to set struct fields from map values.
func (m Map) setFieldsFromMap(sv reflect.Value, fieldMap map[string]int) {
	mi := m.rv.MapRange()
	for mi.Next() {
		key := mi.Key().String()
		if idx, ok := fieldMap[key]; ok {
			fv := sv.Field(idx)
			if fv.CanSet() {
				val := mi.Value()
				// Unwrap interface values
				for val.Kind() == reflect.Interface && !val.IsNil() {
					val = val.Elem()
				}
				if !val.IsValid() {
					continue
				}
				if val.Type().AssignableTo(fv.Type()) {
					fv.Set(val)
				} else if val.Type().ConvertibleTo(fv.Type()) {
					fv.Set(val.Convert(fv.Type()))
				}
			}
		}
	}
}

// ============================================================================
// Entry
// ============================================================================

// Entry represents a map entry with key and value.
// Created via Map.Get() or during iteration.
//
// Example:
//
//	entry := m.Get("a")
//	key, _ := refl.Get[string](entry.Key())  // "a"
//	val, _ := refl.Get[int](entry)           // 1
type Entry struct {
	key     reflect.Value
	val     reflect.Value
	keyType reflect.Type
	valType reflect.Type
}

// reflectValue implements Valuable - returns the value part.
func (e Entry) reflectValue() reflect.Value {
	return e.val
}

// --- Metadata ---

// Key returns the entry's key as a Value.
//
// Example:
//
//	key, _ := refl.Get[string](entry.Key())
func (e Entry) Key() Value {
	return Value{rv: e.key}
}

// KeyInterface returns the key as any.
func (e Entry) KeyInterface() any {
	return e.key.Interface()
}

// KeyType returns the key's type.
func (e Entry) KeyType() reflect.Type {
	return e.keyType
}

// ValType returns the value's type.
func (e Entry) ValType() reflect.Type {
	return e.valType
}

// --- Value Operations ---

// Kind returns the value's kind.
func (e Entry) Kind() Kind {
	return e.val.Kind()
}

// Interface returns the entry value as any.
func (e Entry) Interface() any {
	return e.val.Interface()
}

// IsNil returns true if the value is nil.
func (e Entry) IsNil() bool {
	switch e.val.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return e.val.IsNil()
	default:
		return false
	}
}

// IsZero returns true if the value is zero.
func (e Entry) IsZero() bool {
	return e.val.IsZero()
}

// Value returns the entry value as a Value.
func (e Entry) Value() Value {
	return Value{rv: e.val}
}

// --- Navigation ---

// Struct returns the entry value as a Struct.
// Panics if not a struct.
//
// Example:
//
//	// For map[string]User
//	user := m.Get("alice").Struct()
//	age, _ := refl.Get[int](user.Field("Age"))
func (e Entry) Struct() Struct {
	rv := deref(e.val)
	if rv.Kind() != reflect.Struct {
		panic(fmt.Sprintf("gref: map value is %s, not struct", rv.Kind()))
	}
	return Struct{rv: rv, rt: rv.Type()}
}

// Slice returns the entry value as a Slice.
// Panics if not a slice.
func (e Entry) Slice() Slice {
	rv := deref(e.val)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		panic(fmt.Sprintf("gref: map value is %s, not slice", rv.Kind()))
	}
	return Slice{rv: rv, elemType: rv.Type().Elem()}
}

// Map returns the entry value as a Map.
// Panics if not a map.
func (e Entry) Map() Map {
	rv := deref(e.val)
	if rv.Kind() != reflect.Map {
		panic(fmt.Sprintf("gref: map value is %s, not map", rv.Kind()))
	}
	return Map{rv: rv, keyType: rv.Type().Key(), valType: rv.Type().Elem()}
}

// Ptr returns the entry value as a Ptr.
// Panics if not a pointer.
func (e Entry) Ptr() Ptr {
	if e.val.Kind() != reflect.Pointer {
		panic(fmt.Sprintf("gref: map value is %s, not pointer", e.val.Kind()))
	}
	return Ptr{rv: e.val, elemType: e.val.Type().Elem()}
}
