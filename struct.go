package gref

import (
	"fmt"
	"iter"
	"reflect"
	"strings"
)

// Struct provides operations on struct values.
// Create via Value.Struct().
//
// Example:
//
//	type User struct {
//	    Name string `json:"name" validate:"required"`
//	    Age  int    `json:"age"`
//	}
//
//	s := refl.From(&user).Struct()
//	name, _ := refl.Get[string](s.Field("Name"))
//	tag := s.Field("Name").Tag("json")  // "name"
type Struct struct {
	rv reflect.Value
	rt reflect.Type
}

// reflectValue implements Valuable.
func (s Struct) reflectValue() reflect.Value {
	return s.rv
}

// --- Field Access ---

// Field returns a Field by name.
// Supports dot notation for nested access: "Address.City"
// Panics if the field is not found.
//
// Example:
//
//	f := s.Field("Name")           // direct field
//	f := s.Field("Address.City")   // nested via dot notation
func (s Struct) Field(name string) Field {
	// Handle dot notation for nested access
	if idx := strings.Index(name, "."); idx != -1 {
		first := name[:idx]
		rest := name[idx+1:]
		return s.Field(first).Struct().Field(rest)
	}

	sf, ok := s.rt.FieldByName(name)
	if !ok {
		panic(fmt.Sprintf("gref: field %q not found in %s", name, s.rt))
	}
	return Field{rv: s.rv.FieldByName(name), sf: sf}
}

// TryField returns a Field by name wrapped in an Option.
// Returns None if the field is not found.
// Supports dot notation for nested access.
//
// Example:
//
//	field := s.TryField("Name").Or(defaultField)
//	if field, ok := s.TryField("Name").Value(); ok { ... }
func (s Struct) TryField(name string) Option[Field] {
	// Handle dot notation
	if idx := strings.Index(name, "."); idx != -1 {
		first := name[:idx]
		rest := name[idx+1:]
		f, ok := s.TryField(first).Value()
		if !ok {
			return None[Field]()
		}
		rv := deref(f.rv)
		if rv.Kind() != reflect.Struct {
			return None[Field]()
		}
		nested := Struct{rv: rv, rt: rv.Type()}
		return nested.TryField(rest)
	}

	sf, ok := s.rt.FieldByName(name)
	if !ok {
		return None[Field]()
	}
	return Some(Field{rv: s.rv.FieldByName(name), sf: sf})
}

// FieldByIndex returns a Field by its index path.
// Useful for embedded fields.
//
// Example:
//
//	f := s.FieldByIndex([]int{0})       // first field
//	f := s.FieldByIndex([]int{1, 0})    // first field of embedded struct at index 1
func (s Struct) FieldByIndex(index []int) Field {
	sf := s.rt.FieldByIndex(index)
	fv := s.rv.FieldByIndex(index)
	return Field{rv: fv, sf: sf}
}

// --- Field Modification ---

// SetField sets a field value by name.
// Panics if field not found or not settable.
//
// Example:
//
//	s := refl.From(&user).Struct()
//	s.SetField("Name", "Bob")
func (s Struct) SetField(name string, value any) {
	s.Field(name).Set(value)
}

// --- Field Information ---

// NumFields returns the number of fields.
func (s Struct) NumFields() int {
	return s.rt.NumField()
}

// Type returns the struct's reflect.Type.
func (s Struct) Type() reflect.Type {
	return s.rt
}

// --- Field Iteration ---

// Fields returns an iterator over all fields.
//
// Example:
//
//	s.Fields().Each(func(f refl.Field) bool {
//	    fmt.Println(f.Name(), f.Tag("json"))
//	    return true  // continue
//	})
func (s Struct) Fields() FieldIter {
	return FieldIter{s: s, count: s.rt.NumField()}
}

// --- Methods ---

// Method returns a method by name.
// Looks for methods on both value and pointer receivers.
// Panics if the method is not found.
//
// Example:
//
//	f := s.Method("String")
//	result := f.Call()
func (s Struct) Method(name string) Func {
	// Try value receiver first
	m := s.rv.MethodByName(name)
	if !m.IsValid() && s.rv.CanAddr() {
		// Try pointer receiver
		m = s.rv.Addr().MethodByName(name)
	}
	if !m.IsValid() {
		panic(fmt.Sprintf("gref: method %q not found on %s", name, s.rt))
	}
	return Func{rv: m, rt: m.Type()}
}

// TryMethod returns a method by name wrapped in an Option.
// Returns None if not found.
//
//	method := s.TryMethod("String").Or(defaultMethod)
//	if method, ok := s.TryMethod("String").Value(); ok { ... }
func (s Struct) TryMethod(name string) Option[Func] {
	m := s.rv.MethodByName(name)
	if !m.IsValid() && s.rv.CanAddr() {
		m = s.rv.Addr().MethodByName(name)
	}
	if !m.IsValid() {
		return None[Func]()
	}
	return Some(Func{rv: m, rt: m.Type()})
}

// NumMethods returns the number of exported methods.
func (s Struct) NumMethods() int {
	return s.rv.NumMethod()
}

// --- Conversion ---

// ToMap converts the struct to map[string]any using field names as keys.
//
// Example:
//
//	m := s.ToMap()  // {"Name": "Alice", "Age": 30}
func (s Struct) ToMap() map[string]any {
	result := make(map[string]any, s.rt.NumField())

	for i := 0; i < s.rt.NumField(); i++ {
		sf := s.rt.Field(i)
		if !sf.IsExported() {
			continue
		}

		fv := s.rv.Field(i)
		if fv.CanInterface() {
			result[sf.Name] = fv.Interface()
		}
	}

	return result
}

// ToMapTag converts the struct to map[string]any using tag values as keys.
// Fields tagged with "-" are skipped.
//
// Example:
//
//	m := s.ToMapTag("json")  // {"name": "Alice", "age": 30}
func (s Struct) ToMapTag(tagName string) map[string]any {
	result := make(map[string]any, s.rt.NumField())

	for i := 0; i < s.rt.NumField(); i++ {
		sf := s.rt.Field(i)
		if !sf.IsExported() {
			continue
		}

		fv := s.rv.Field(i)
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

		if fv.CanInterface() {
			result[key] = fv.Interface()
		}
	}

	return result
}

// --- Interface Check ---

// Implements checks if the struct implements an interface.
// Pass a nil pointer of the interface type: (*MyInterface)(nil)
//
// Example:
//
//	if s.Implements((*fmt.Stringer)(nil)) {
//	    // struct has String() method
//	}
func (s Struct) Implements(iface any) bool {
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
	return s.rt.Implements(ifaceType) || reflect.PointerTo(s.rt).Implements(ifaceType)
}

// Value returns this struct as a Value.
func (s Struct) Value() Value {
	return Value{rv: s.rv}
}

// Interface returns the struct as any.
func (s Struct) Interface() any {
	return s.rv.Interface()
}

// ============================================================================
// Field
// ============================================================================

// Field represents a struct field with both its value and metadata.
// Created via Struct.Field().
//
// Example:
//
//	f := s.Field("Name")
//	tag := f.Tag("json")           // "name"
//	name, _ := refl.Get[string](f) // "Alice"
type Field struct {
	rv reflect.Value
	sf reflect.StructField
}

// reflectValue implements Valuable.
func (f Field) reflectValue() reflect.Value {
	return f.rv
}

// --- Metadata ---

// Name returns the field name.
func (f Field) Name() string {
	return f.sf.Name
}

// Tag returns the value of a struct tag key.
//
// Example:
//
//	tag := f.Tag("json")      // "name"
//	tag := f.Tag("validate")  // "required"
func (f Field) Tag(key string) string {
	return f.sf.Tag.Get(key)
}

// TagLookup returns the tag value and whether it exists.
func (f Field) TagLookup(key string) (string, bool) {
	return f.sf.Tag.Lookup(key)
}

// RawTag returns the entire struct tag string.
func (f Field) RawTag() reflect.StructTag {
	return f.sf.Tag
}

// ParsedTag returns a parsed struct tag with name and options separated.
// For tags like `json:"name,omitempty"`, returns {Name: "name", Options: ["omitempty"]}
//
// Example:
//
//	pt := f.ParsedTag("json")
//	pt.Name                    // "name"
//	pt.HasOption("omitempty")  // true
func (f Field) ParsedTag(key string) ParsedTag {
	val, ok := f.sf.Tag.Lookup(key)
	if !ok {
		return ParsedTag{Exists: false}
	}

	pt := ParsedTag{Exists: true, Raw: val}
	parts := strings.Split(val, ",")
	if len(parts) > 0 {
		pt.Name = parts[0]
		pt.Options = parts[1:]
	}
	return pt
}

// Type returns the field's type.
func (f Field) Type() reflect.Type {
	return f.sf.Type
}

// Index returns the field's index in the struct.
func (f Field) Index() []int {
	return f.sf.Index
}

// IsExported returns true if the field is exported.
func (f Field) IsExported() bool {
	return f.sf.IsExported()
}

// IsEmbedded returns true if the field is embedded (anonymous).
func (f Field) IsEmbedded() bool {
	return f.sf.Anonymous
}

// Offset returns the field's offset in bytes within the struct.
func (f Field) Offset() uintptr {
	return f.sf.Offset
}

// StructField returns the underlying reflect.StructField.
func (f Field) StructField() reflect.StructField {
	return f.sf
}

// --- Value Operations ---

// Kind returns the field value's kind.
func (f Field) Kind() Kind {
	return f.rv.Kind()
}

// IsNil returns true if the field value is nil.
func (f Field) IsNil() bool {
	switch f.rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return f.rv.IsNil()
	default:
		return false
	}
}

// IsZero returns true if the field value is zero.
func (f Field) IsZero() bool {
	return f.rv.IsZero()
}

// Interface returns the field value as any.
// Panics if the field is unexported.
func (f Field) Interface() any {
	if !f.rv.CanInterface() {
		panic(fmt.Sprintf("gref: cannot access unexported field %q", f.sf.Name))
	}
	return f.rv.Interface()
}

// CanSet returns true if the field can be modified.
func (f Field) CanSet() bool {
	return f.rv.CanSet()
}

// Set sets the field value.
// Panics if not settable.
//
// Example:
//
//	s.Field("Name").Set("Bob")
func (f Field) Set(value any) {
	if !f.rv.CanSet() {
		panic(fmt.Sprintf("gref: field %q is not settable (pass pointer to struct?)", f.sf.Name))
	}
	f.rv.Set(reflect.ValueOf(value))
}

// Value returns the field as a Value.
func (f Field) Value() Value {
	return Value{rv: f.rv}
}

// Addr returns a pointer to the field value.
// Useful for sql.Scan and similar APIs that need a pointer destination.
//
// Example:
//
//	dest[i] = field.Addr().Interface()  // for sql.Scan
func (f Field) Addr() Ptr {
	if !f.rv.CanAddr() {
		panic(fmt.Sprintf("gref: field %q is not addressable", f.sf.Name))
	}
	return Ptr{rv: f.rv.Addr(), elemType: f.rv.Type()}
}

// --- Navigation (when field value is a composite type) ---

// Struct returns the field value as a Struct.
// Automatically dereferences pointers.
// Panics if not a struct.
//
// Example:
//
//	city, _ := refl.Get[string](s.Field("Address").Struct().Field("City"))
func (f Field) Struct() Struct {
	rv := deref(f.rv)
	if rv.Kind() != reflect.Struct {
		panic(fmt.Sprintf("gref: field %q is %s, not struct", f.sf.Name, rv.Kind()))
	}
	return Struct{rv: rv, rt: rv.Type()}
}

// Slice returns the field value as a Slice.
// Automatically dereferences pointers.
// Panics if not a slice or array.
func (f Field) Slice() Slice {
	rv := deref(f.rv)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		panic(fmt.Sprintf("gref: field %q is %s, not slice", f.sf.Name, rv.Kind()))
	}
	return Slice{rv: rv, elemType: rv.Type().Elem()}
}

// Map returns the field value as a Map.
// Automatically dereferences pointers.
// Panics if not a map.
func (f Field) Map() Map {
	rv := deref(f.rv)
	if rv.Kind() != reflect.Map {
		panic(fmt.Sprintf("gref: field %q is %s, not map", f.sf.Name, rv.Kind()))
	}
	return Map{rv: rv, keyType: rv.Type().Key(), valType: rv.Type().Elem()}
}

// Chan returns the field value as a Chan.
// Panics if not a channel.
func (f Field) Chan() Chan {
	rv := deref(f.rv)
	if rv.Kind() != reflect.Chan {
		panic(fmt.Sprintf("gref: field %q is %s, not chan", f.sf.Name, rv.Kind()))
	}
	return Chan{rv: rv, elemType: rv.Type().Elem(), dir: rv.Type().ChanDir()}
}

// Func returns the field value as a Func.
// Panics if not a function.
func (f Field) Func() Func {
	rv := deref(f.rv)
	if rv.Kind() != reflect.Func {
		panic(fmt.Sprintf("gref: field %q is %s, not func", f.sf.Name, rv.Kind()))
	}
	return Func{rv: rv, rt: rv.Type()}
}

// Ptr returns the field value as a Ptr.
// Does NOT auto-dereference.
// Panics if not a pointer.
func (f Field) Ptr() Ptr {
	if f.rv.Kind() != reflect.Pointer {
		panic(fmt.Sprintf("gref: field %q is %s, not pointer", f.sf.Name, f.rv.Kind()))
	}
	return Ptr{rv: f.rv, elemType: f.rv.Type().Elem()}
}

// ============================================================================
// ParsedTag
// ============================================================================

// ParsedTag represents a parsed struct tag value.
type ParsedTag struct {
	Exists  bool     // Whether the tag key exists
	Raw     string   // The raw tag value
	Name    string   // The primary name (before first comma)
	Options []string // Options after the first comma
}

// HasOption checks if an option is present.
//
// Example:
//
//	pt := f.ParsedTag("json")  // "name,omitempty"
//	pt.HasOption("omitempty")  // true
func (t ParsedTag) HasOption(opt string) bool {
	for _, o := range t.Options {
		if o == opt {
			return true
		}
	}
	return false
}

// ============================================================================
// FieldIter
// ============================================================================

// FieldIter iterates over struct fields.
type FieldIter struct {
	s     Struct
	count int
}

// Each calls fn for each field. Return false to stop iteration.
//
// Example:
//
//	s.Fields().Each(func(f refl.Field) bool {
//	    fmt.Println(f.Name(), f.Tag("json"))
//	    return true  // continue
//	})
func (it FieldIter) Each(fn func(Field) bool) {
	for i := 0; i < it.count; i++ {
		sf := it.s.rt.Field(i)
		fv := it.s.rv.Field(i)
		if !fn(Field{rv: fv, sf: sf}) {
			break
		}
	}
}

// Collect returns all fields as a slice.
func (it FieldIter) Collect() []Field {
	fields := make([]Field, it.count)
	for i := 0; i < it.count; i++ {
		sf := it.s.rt.Field(i)
		fv := it.s.rv.Field(i)
		fields[i] = Field{rv: fv, sf: sf}
	}
	return fields
}

// Iter returns an iterator for use with range.
//
// Example:
//
//	for field := range s.Fields().Iter() {
//	    fmt.Println(field.Name(), field.Tag("json"))
//	}
func (it FieldIter) Iter() iter.Seq[Field] {
	return func(yield func(Field) bool) {
		for i := 0; i < it.count; i++ {
			sf := it.s.rt.Field(i)
			fv := it.s.rv.Field(i)
			if !yield(Field{rv: fv, sf: sf}) {
				return
			}
		}
	}
}

// Exported returns an iterator over exported fields only.
func (it FieldIter) Exported() FilteredFieldIter {
	return FilteredFieldIter{
		iter: it,
		pred: func(f Field) bool { return f.IsExported() },
	}
}

// WithTag returns an iterator over fields with the specified tag.
//
// Example:
//
//	s.Fields().WithTag("validate").Each(func(f refl.Field) bool {
//	    fmt.Println(f.Name(), "requires validation")
//	    return true
//	})
func (it FieldIter) WithTag(key string) FilteredFieldIter {
	return FilteredFieldIter{
		iter: it,
		pred: func(f Field) bool {
			_, ok := f.TagLookup(key)
			return ok
		},
	}
}

// FilteredFieldIter is a filtered field iterator.
type FilteredFieldIter struct {
	iter FieldIter
	pred func(Field) bool
}

// Each calls fn for each matching field.
func (it FilteredFieldIter) Each(fn func(Field) bool) {
	it.iter.Each(func(f Field) bool {
		if it.pred(f) {
			return fn(f)
		}
		return true
	})
}

// Collect returns all matching fields as a slice.
func (it FilteredFieldIter) Collect() []Field {
	var fields []Field
	it.Each(func(f Field) bool {
		fields = append(fields, f)
		return true
	})
	return fields
}

// Iter returns an iterator for use with range.
//
// Example:
//
//	for field := range s.Fields().Exported().Iter() {
//	    fmt.Println(field.Name())
//	}
func (it FilteredFieldIter) Iter() iter.Seq[Field] {
	return func(yield func(Field) bool) {
		it.Each(func(f Field) bool {
			return yield(f)
		})
	}
}
