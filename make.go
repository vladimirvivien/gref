package gref

import (
	"fmt"
	"reflect"
)

// ============================================================================
// Predefined Types (for runtime type definitions)
// ============================================================================

var (
	// Basic types
	BoolType    = reflect.TypeOf(false)
	StringType  = reflect.TypeOf("")
	IntType     = reflect.TypeOf(0)
	Int8Type    = reflect.TypeOf(int8(0))
	Int16Type   = reflect.TypeOf(int16(0))
	Int32Type   = reflect.TypeOf(int32(0))
	Int64Type   = reflect.TypeOf(int64(0))
	UintType    = reflect.TypeOf(uint(0))
	Uint8Type   = reflect.TypeOf(uint8(0))
	Uint16Type  = reflect.TypeOf(uint16(0))
	Uint32Type  = reflect.TypeOf(uint32(0))
	Uint64Type  = reflect.TypeOf(uint64(0))
	UintptrType = reflect.TypeOf(uintptr(0))
	Float32Type = reflect.TypeOf(float32(0))
	Float64Type = reflect.TypeOf(float64(0))
	ByteType    = Uint8Type
	RuneType    = Int32Type

	// Common composite types
	ByteSliceType   = reflect.TypeOf([]byte(nil))
	StringSliceType = reflect.TypeOf([]string(nil))
	AnyType         = reflect.TypeOf((*any)(nil)).Elem()
	ErrorType       = reflect.TypeOf((*error)(nil)).Elem()
)

// ============================================================================
// COMPILE-TIME VALUE CREATION: Make[T]()
// ============================================================================

// Maker is returned by Make[T]() and provides type-appropriate operations.
// It creates refl wrapper types for compile-time known Go types.
//
// For non-function types, use navigation methods to get specialized types:
//
//	refl.Make[[]int]().Slice(0, 10).Append(1, 2, 3)
//	refl.Make[map[string]int]().Map().Set("key", 42)
//	refl.Make[chan string]().Chan(5).Send("hello")
//	refl.Make[User]().Struct().SetField("Name", "Alice")
//
// For function types, use Impl() to provide the implementation:
//
//	refl.Make[func(string) bool]().Impl(func(s string) bool { return len(s) > 0 })
type Maker[T any] struct {
	rt reflect.Type
}

// Make creates a Maker for type T.
// Use navigation methods (.Slice(), .Map(), .Chan(), .Struct()) to create the actual value.
//
// Examples:
//
//	sl := refl.Make[[]int]().Slice(0, 10)       // Slice with len=0, cap=10
//	m := refl.Make[map[string]int]().Map()     // Empty map
//	c := refl.Make[chan string]().Chan(5)      // Buffered channel
//	s := refl.Make[User]().Struct()            // Zero-value struct
//	f := refl.Make[func(int) bool]().Impl(fn)  // Function with impl
func Make[T any]() Maker[T] {
	var zero T
	t := reflect.TypeOf(zero)
	
	// Handle interface types (like any)
	if t == nil {
		t = reflect.TypeOf((*T)(nil)).Elem()
	}

	return Maker[T]{rt: t}
}

// --- Navigation Methods ---

// Slice returns this maker's value as a Slice.
// Arguments: (length, capacity) - both optional, default to 0.
// Panics if T is not a slice type.
//
//	sl := refl.Make[[]int]().Slice(0, 10)  // len=0, cap=10
//	sl := refl.Make[[]int]().Slice(5)      // len=5, cap=5
//	sl := refl.Make[[]int]().Slice()       // len=0, cap=0
func (m Maker[T]) Slice(args ...int) Slice {
	if m.rt.Kind() != reflect.Slice {
		panic(fmt.Sprintf("gref: Make[%s].Slice() requires slice type", m.rt))
	}
	length := 0
	capacity := 0
	if len(args) > 0 {
		length = args[0]
	}
	if len(args) > 1 {
		capacity = args[1]
	}
	if capacity < length {
		capacity = length
	}
	rv := reflect.MakeSlice(m.rt, length, capacity)
	return Slice{rv: rv, elemType: m.rt.Elem()}
}

// Map returns this maker's value as a Map.
// Panics if T is not a map type.
//
//	m := refl.Make[map[string]int]().Map()
func (m Maker[T]) Map() Map {
	if m.rt.Kind() != reflect.Map {
		panic(fmt.Sprintf("gref: Make[%s].Map() requires map type", m.rt))
	}
	rv := reflect.MakeMap(m.rt)
	return Map{rv: rv, keyType: m.rt.Key(), valType: m.rt.Elem()}
}

// Chan returns this maker's value as a Chan.
// Argument: bufferSize - optional, defaults to 0 (unbuffered).
// Panics if T is not a channel type.
//
//	c := refl.Make[chan string]().Chan(5)   // buffered
//	c := refl.Make[chan int]().Chan()       // unbuffered
func (m Maker[T]) Chan(args ...int) Chan {
	if m.rt.Kind() != reflect.Chan {
		panic(fmt.Sprintf("gref: Make[%s].Chan() requires chan type", m.rt))
	}
	bufSize := 0
	if len(args) > 0 {
		bufSize = args[0]
	}
	rv := reflect.MakeChan(m.rt, bufSize)
	return Chan{rv: rv, elemType: m.rt.Elem(), dir: m.rt.ChanDir()}
}

// Struct returns this maker's value as a Struct.
// Panics if T is not a struct type.
//
//	s := refl.Make[User]().Struct()
func (m Maker[T]) Struct() Struct {
	if m.rt.Kind() != reflect.Struct {
		panic(fmt.Sprintf("gref: Make[%s].Struct() requires struct type", m.rt))
	}
	// Create settable struct via pointer
	rv := reflect.New(m.rt).Elem()
	return Struct{rv: rv, rt: m.rt}
}

// Ptr returns this maker's value as a Ptr.
// Panics if T is not a pointer type.
//
//	p := refl.Make[*User]().Ptr()
func (m Maker[T]) Ptr() Ptr {
	if m.rt.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("gref: Make[%s].Ptr() requires pointer type", m.rt))
	}
	rv := reflect.New(m.rt.Elem())
	return Ptr{rv: rv, elemType: m.rt.Elem()}
}

// --- Function Implementation ---

// Impl provides the implementation for a function type.
// Panics if T is not a function type.
//
//	f := refl.Make[func(string, int) bool]().Impl(
//	    func(name string, age int) bool {
//	        return age >= 18
//	    },
//	)
func (m Maker[T]) Impl(fn T) Func {
	if m.rt.Kind() != reflect.Func {
		panic(fmt.Sprintf("gref: Make[%s].Impl() requires func type", m.rt))
	}
	rv := reflect.ValueOf(fn)
	if !rv.IsValid() || rv.Kind() != reflect.Func {
		panic("gref: Impl requires a function")
	}
	return Func{rv: rv, rt: m.rt}
}

// ImplDynamic provides a dynamic implementation using Value-based callback.
// Useful when you need to work with reflection inside the function.
// Panics if T is not a function type.
func (m Maker[T]) ImplDynamic(impl func(args []Value) []Value) Func {
	if m.rt.Kind() != reflect.Func {
		panic(fmt.Sprintf("gref: Make[%s].ImplDynamic() requires func type", m.rt))
	}
	wrapper := func(in []reflect.Value) []reflect.Value {
		args := make([]Value, len(in))
		for i, rv := range in {
			args[i] = Value{rv: rv}
		}
		results := impl(args)
		out := make([]reflect.Value, len(results))
		for i, v := range results {
			out[i] = v.rv
		}
		return out
	}
	rv := reflect.MakeFunc(m.rt, wrapper)
	return Func{rv: rv, rt: m.rt}
}

// --- General Methods ---

// Type returns the type.
func (m Maker[T]) Type() Type {
	return m.rt
}

// ============================================================================
// LEGACY COMPILE-TIME FUNCTIONS (kept for compatibility)
// These delegate to Make[T]() internally.
// ============================================================================

// MakeSlice creates a new slice with compile-time known element type.
// Deprecated: Use Make[[]T]().Slice(length, capacity) instead.
func MakeSlice[T any](length, capacity int) Slice {
	return Make[[]T]().Slice(length, capacity)
}

// MakeMap creates a new map with compile-time known types.
// Deprecated: Use Make[map[K]V]().Map() instead.
func MakeMap[K comparable, V any]() Map {
	var zeroK K
	var zeroV V
	keyType := reflect.TypeOf(zeroK)
	valType := reflect.TypeOf(zeroV)
	if keyType == nil {
		keyType = AnyType
	}
	if valType == nil {
		valType = AnyType
	}
	rv := reflect.MakeMap(reflect.MapOf(keyType, valType))
	return Map{rv: rv, keyType: keyType, valType: valType}
}

// MakeChan creates a new channel with compile-time known element type.
// Deprecated: Use Make[chan T]().Chan(bufferSize) instead.
func MakeChan[T any](bufferSize int) Chan {
	return Make[chan T]().Chan(bufferSize)
}

// MakeStruct creates a new settable zero-value instance of a compile-time struct type.
// Deprecated: Use Make[T]().Struct() instead.
func MakeStruct[T any]() Struct {
	var zero T
	t := reflect.TypeOf(zero)
	if t == nil {
		panic("gref: MakeStruct requires a concrete type")
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		panic(fmt.Sprintf("gref: MakeStruct requires struct type, got %s", t.Kind()))
	}
	rv := reflect.New(t).Elem()
	return Struct{rv: rv, rt: t}
}

// MakeFunc creates a function maker for compile-time known function signature.
// Deprecated: Use Make[F]().Impl(fn) instead.
func MakeFunc[F any]() FuncMaker[F] {
	var zero F
	t := reflect.TypeOf(zero)
	if t == nil || t.Kind() != reflect.Func {
		panic("gref: MakeFunc requires a function type parameter")
	}
	return FuncMaker[F]{funcType: t}
}

// FuncMaker is returned by MakeFunc to allow setting the implementation.
// Deprecated: Use Make[F]().Impl(fn) instead.
type FuncMaker[F any] struct {
	funcType reflect.Type
}

// Impl provides the implementation for the function.
func (fm FuncMaker[F]) Impl(fn F) Func {
	rv := reflect.ValueOf(fn)
	if !rv.IsValid() || rv.Kind() != reflect.Func {
		panic("gref: Impl requires a function")
	}
	return Func{rv: rv, rt: fm.funcType}
}

// ImplDynamic provides a dynamic implementation using Value-based callback.
func (fm FuncMaker[F]) ImplDynamic(impl func(args []Value) []Value) Func {
	wrapper := func(in []reflect.Value) []reflect.Value {
		args := make([]Value, len(in))
		for i, rv := range in {
			args[i] = Value{rv: rv}
		}
		results := impl(args)
		out := make([]reflect.Value, len(results))
		for i, v := range results {
			out[i] = v.rv
		}
		return out
	}
	rv := reflect.MakeFunc(fm.funcType, wrapper)
	return Func{rv: rv, rt: fm.funcType}
}

// ============================================================================
// RUNTIME VALUE CREATION: Def()
// Uses reflect.Type for dynamic type construction at runtime.
// ============================================================================

// RuntimeDef is returned by Def() and provides runtime type construction.
type RuntimeDef struct{}

// Def returns a RuntimeDef for creating values with runtime-determined types.
//
// Examples:
//
//	sl := refl.Def().Slice(refl.IntType, 0, 10)
//	m := refl.Def().Map(refl.StringType, refl.IntType)
//	c := refl.Def().Chan(refl.StringType, 5)
//	s := refl.Def().Struct("Person").Field(...).Struct()
//	f := refl.Def().Func("isAdult").Arg(...).Return(...).Impl(...)
func Def() RuntimeDef {
	return RuntimeDef{}
}

// Slice creates a new slice with runtime-determined element type.
//
//	sl := refl.Def().Slice(refl.IntType, 0, 10)
//	sl = sl.Append(1, 2, 3)
func (RuntimeDef) Slice(elemType Type, length, capacity int) Slice {
	if elemType == nil {
		panic("gref: Def().Slice() requires element type")
	}
	rv := reflect.MakeSlice(reflect.SliceOf(elemType), length, capacity)
	return Slice{rv: rv, elemType: elemType}
}

// Map creates a new map with runtime-determined types.
//
//	m := refl.Def().Map(refl.StringType, refl.IntType)
//	m = m.Set("key", 42)
func (RuntimeDef) Map(keyType, valType Type) Map {
	if keyType == nil || valType == nil {
		panic("gref: Def().Map() requires key and value types")
	}
	rv := reflect.MakeMap(reflect.MapOf(keyType, valType))
	return Map{rv: rv, keyType: keyType, valType: valType}
}

// Chan creates a new channel with runtime-determined element type.
//
//	c := refl.Def().Chan(refl.StringType, 5)
//	c.Send("hello")
func (RuntimeDef) Chan(elemType Type, bufferSize int) Chan {
	if elemType == nil {
		panic("gref: Def().Chan() requires element type")
	}
	rv := reflect.MakeChan(reflect.ChanOf(reflect.BothDir, elemType), bufferSize)
	return Chan{rv: rv, elemType: elemType, dir: BothDir}
}

// Struct creates a new struct builder for runtime struct definition.
//
//	s := refl.Def().Struct("Person").
//	    Field(refl.FieldDef{Name: "Name", Type: refl.StringType}).
//	    Field(refl.FieldDef{Name: "Age", Type: refl.IntType}).
//	    Set("Name", "Alice").
//	    Struct()
func (RuntimeDef) Struct(name string) *StructBuilder {
	return &StructBuilder{name: name}
}

// Func creates a new function builder for runtime function definition.
//
//	f := refl.Def().Func("isAdult").
//	    Arg(refl.ArgDef{Type: refl.StringType}).
//	    Arg(refl.ArgDef{Type: refl.IntType}).
//	    Return(refl.ReturnDef{Type: refl.BoolType}).
//	    Impl(func(args []refl.Value) []refl.Value {
//	        age, _ := refl.Get[int](args[1])
//	        return []refl.Value{refl.From(age >= 18)}
//	    })
func (RuntimeDef) Func(name string) *FuncBuilder {
	return &FuncBuilder{name: name}
}

// ----------------------------------------------------------------------------
// StructBuilder - Runtime struct definition
// ----------------------------------------------------------------------------

// FieldDef describes a struct field for runtime struct definition.
type FieldDef struct {
	Name string
	Type Type
	Tag  string // e.g., `json:"name"`
}

// StructBuilder builds struct types and values at runtime.
type StructBuilder struct {
	name   string
	fields []reflect.StructField
	values map[string]any
	rv     reflect.Value
	rt     reflect.Type
	built  bool
}

// Field adds a field definition to the struct.
// Panics if the struct has already been finalized.
func (sb *StructBuilder) Field(f FieldDef) *StructBuilder {
	if sb.built {
		panic("gref: cannot add fields after struct is finalized")
	}
	sf := reflect.StructField{
		Name: f.Name,
		Type: f.Type,
		Tag:  reflect.StructTag(f.Tag),
	}
	sb.fields = append(sb.fields, sf)
	return sb
}

// Set sets a field value by name.
// Values are stored and applied when Struct() is called.
func (sb *StructBuilder) Set(name string, value any) *StructBuilder {
	if sb.values == nil {
		sb.values = make(map[string]any)
	}
	sb.values[name] = value
	return sb
}

// finalize builds the struct type if not already built.
func (sb *StructBuilder) finalize() {
	if sb.built {
		return
	}
	if len(sb.fields) == 0 {
		panic("gref: StructBuilder has no fields defined")
	}
	sb.rt = reflect.StructOf(sb.fields)
	sb.rv = reflect.New(sb.rt).Elem()
	sb.built = true

	// Apply stored values
	for name, value := range sb.values {
		f := sb.rv.FieldByName(name)
		if !f.IsValid() {
			panic(fmt.Sprintf("gref: field %q not found", name))
		}
		if !f.CanSet() {
			panic(fmt.Sprintf("gref: field %q is not settable", name))
		}
		val := reflect.ValueOf(value)
		if !val.Type().AssignableTo(f.Type()) {
			panic(fmt.Sprintf("gref: cannot assign %s to field %q of type %s", val.Type(), name, f.Type()))
		}
		f.Set(val)
	}
}

// Struct returns the finalized Struct.
// This replaces the old Build() method.
func (sb *StructBuilder) Struct() Struct {
	sb.finalize()
	return Struct{rv: sb.rv, rt: sb.rt}
}

// Interface returns the struct value as any.
func (sb *StructBuilder) Interface() any {
	sb.finalize()
	return sb.rv.Interface()
}

// Ptr returns a pointer to the struct.
func (sb *StructBuilder) Ptr() any {
	sb.finalize()
	ptr := reflect.New(sb.rt)
	ptr.Elem().Set(sb.rv)
	return ptr.Interface()
}

// Type returns the struct's type.
func (sb *StructBuilder) Type() Type {
	sb.finalize()
	return sb.rt
}

// ----------------------------------------------------------------------------
// FuncBuilder - Runtime function definition
// ----------------------------------------------------------------------------

// ArgDef describes a function argument.
type ArgDef struct {
	Name string // optional, for documentation
	Type Type
}

// ReturnDef describes a function return value.
type ReturnDef struct {
	Name string // optional, for documentation
	Type Type
}

// FuncBuilder builds function types and values at runtime.
type FuncBuilder struct {
	name     string
	args     []reflect.Type
	returns  []reflect.Type
	variadic bool
}

// Arg adds an argument definition.
func (fb *FuncBuilder) Arg(a ArgDef) *FuncBuilder {
	fb.args = append(fb.args, a.Type)
	return fb
}

// Return adds a return value definition.
func (fb *FuncBuilder) Return(r ReturnDef) *FuncBuilder {
	fb.returns = append(fb.returns, r.Type)
	return fb
}

// Variadic marks the function as variadic (last arg must be slice type).
func (fb *FuncBuilder) Variadic() *FuncBuilder {
	fb.variadic = true
	return fb
}

// Type returns the function type.
func (fb *FuncBuilder) Type() Type {
	return reflect.FuncOf(fb.args, fb.returns, fb.variadic)
}

// Impl sets the implementation and returns the finalized Func.
// This replaces the old Build() method.
func (fb *FuncBuilder) Impl(impl func(args []Value) []Value) Func {
	funcType := reflect.FuncOf(fb.args, fb.returns, fb.variadic)

	wrapper := func(in []reflect.Value) []reflect.Value {
		args := make([]Value, len(in))
		for i, rv := range in {
			args[i] = Value{rv: rv}
		}
		results := impl(args)
		out := make([]reflect.Value, len(results))
		for i, v := range results {
			out[i] = v.rv
		}
		return out
	}

	rv := reflect.MakeFunc(funcType, wrapper)
	return Func{rv: rv, rt: funcType}
}

// ----------------------------------------------------------------------------
// StructFrom - Create from existing struct
// ----------------------------------------------------------------------------

// StructFrom creates a StructBuilder from an existing value.
// This allows modifying a copy of an existing struct.
//
//	user := refl.Make[User]().Struct()
//	copy := refl.StructFrom(user).Set("Name", "Bob").Struct()
func StructFrom(v any) *StructBuilder {
	rv := reflect.ValueOf(v)

	// Handle Struct type from our package
	if s, ok := v.(Struct); ok {
		rv = s.rv
	}

	for rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		panic(fmt.Sprintf("gref: StructFrom requires struct, got %s", rv.Kind()))
	}

	// Create a new settable copy
	newRv := reflect.New(rv.Type()).Elem()
	newRv.Set(rv)

	return &StructBuilder{
		name:  rv.Type().Name(),
		rv:    newRv,
		rt:    rv.Type(),
		built: true,
	}
}

// ============================================================================
// LEGACY RUNTIME FUNCTIONS (kept for compatibility)
// ============================================================================

// StructDef creates a new runtime struct definition.
// Deprecated: Use Def().Struct(name) instead.
func StructDef(name string) *StructDefBuilder {
	return &StructDefBuilder{name: name}
}

// StructDefBuilder is the legacy builder type.
// Deprecated: Use StructBuilder instead.
type StructDefBuilder struct {
	name   string
	fields []reflect.StructField
	rv     reflect.Value
	rt     reflect.Type
	built  bool
}

// FieldDef adds a field (legacy API).
func (sb *StructDefBuilder) FieldDef(f FieldDef) *StructDefBuilder {
	if sb.built {
		panic("gref: cannot add fields after struct is built")
	}
	sf := reflect.StructField{
		Name: f.Name,
		Type: f.Type,
		Tag:  reflect.StructTag(f.Tag),
	}
	sb.fields = append(sb.fields, sf)
	return sb
}

func (sb *StructDefBuilder) build() {
	if sb.built {
		return
	}
	if len(sb.fields) == 0 {
		panic("gref: StructDef has no fields defined")
	}
	sb.rt = reflect.StructOf(sb.fields)
	sb.rv = reflect.New(sb.rt).Elem()
	sb.built = true
}

// Set sets a field value (legacy API).
func (sb *StructDefBuilder) Set(name string, value any) *StructDefBuilder {
	sb.build()
	f := sb.rv.FieldByName(name)
	if !f.IsValid() {
		panic(fmt.Sprintf("gref: field %q not found", name))
	}
	if !f.CanSet() {
		panic(fmt.Sprintf("gref: field %q is not settable", name))
	}
	val := reflect.ValueOf(value)
	if !val.Type().AssignableTo(f.Type()) {
		panic(fmt.Sprintf("gref: cannot assign %s to field %q of type %s", val.Type(), name, f.Type()))
	}
	f.Set(val)
	return sb
}

// Build returns the finalized Struct (legacy API).
func (sb *StructDefBuilder) Build() Struct {
	sb.build()
	return Struct{rv: sb.rv, rt: sb.rt}
}

// StructDefFrom creates a struct definition from an existing value.
// Deprecated: Use StructFrom(v) instead.
func StructDefFrom(v any) *StructDefBuilder {
	rv := reflect.ValueOf(v)

	if s, ok := v.(Struct); ok {
		rv = s.rv
	}

	for rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		panic(fmt.Sprintf("gref: StructDefFrom requires struct, got %s", rv.Kind()))
	}

	newRv := reflect.New(rv.Type()).Elem()
	newRv.Set(rv)

	return &StructDefBuilder{
		name:  rv.Type().Name(),
		rv:    newRv,
		rt:    rv.Type(),
		built: true,
	}
}

// SliceDef creates a new slice with runtime-determined element type.
// Deprecated: Use Def().Slice(elemType, length, capacity) instead.
func SliceDef(elemType Type, length, capacity int) Slice {
	return Def().Slice(elemType, length, capacity)
}

// MapDef creates a new map with runtime-determined types.
// Deprecated: Use Def().Map(keyType, valType) instead.
func MapDef(keyType, valType Type) Map {
	return Def().Map(keyType, valType)
}

// MapDefWithSize creates a map with initial size hint.
// Deprecated: Use Def().Map() - size hints are not exposed.
func MapDefWithSize(keyType, valType Type, size int) Map {
	if keyType == nil || valType == nil {
		panic("gref: MapDefWithSize requires key and value types")
	}
	rv := reflect.MakeMapWithSize(reflect.MapOf(keyType, valType), size)
	return Map{rv: rv, keyType: keyType, valType: valType}
}

// ChanDef creates a new channel with runtime-determined element type.
// Deprecated: Use Def().Chan(elemType, bufferSize) instead.
func ChanDef(elemType Type, bufferSize int) Chan {
	return Def().Chan(elemType, bufferSize)
}

// ChanDefWithDir creates a channel with specific direction.
func ChanDefWithDir(elemType Type, bufferSize int, dir ChanDir) Chan {
	if elemType == nil {
		panic("gref: ChanDefWithDir requires element type")
	}
	rv := reflect.MakeChan(reflect.ChanOf(dir, elemType), bufferSize)
	return Chan{rv: rv, elemType: elemType, dir: dir}
}

// FuncDef creates a function builder for runtime function definition.
// Deprecated: Use Def().Func(name) instead.
func FuncDef(name string) *FuncDefBuilder {
	return &FuncDefBuilder{name: name}
}

// FuncDefBuilder is the legacy function builder.
// Deprecated: Use FuncBuilder instead.
type FuncDefBuilder struct {
	name     string
	args     []reflect.Type
	returns  []reflect.Type
	variadic bool
	impl     func(args []Value) []Value
}

// ArgDef adds an argument (legacy API).
func (fb *FuncDefBuilder) ArgDef(a ArgDef) *FuncDefBuilder {
	fb.args = append(fb.args, a.Type)
	return fb
}

// ReturnDef adds a return (legacy API).
func (fb *FuncDefBuilder) ReturnDef(r ReturnDef) *FuncDefBuilder {
	fb.returns = append(fb.returns, r.Type)
	return fb
}

// Variadic marks variadic (legacy API).
func (fb *FuncDefBuilder) Variadic() *FuncDefBuilder {
	fb.variadic = true
	return fb
}

// Impl sets implementation (legacy API).
func (fb *FuncDefBuilder) Impl(impl func(args []Value) []Value) *FuncDefBuilder {
	fb.impl = impl
	return fb
}

// Type returns function type (legacy API).
func (fb *FuncDefBuilder) Type() Type {
	return reflect.FuncOf(fb.args, fb.returns, fb.variadic)
}

// Build creates the function (legacy API).
func (fb *FuncDefBuilder) Build() Func {
	if fb.impl == nil {
		panic("gref: FuncDef requires implementation via Impl()")
	}

	funcType := reflect.FuncOf(fb.args, fb.returns, fb.variadic)

	wrapper := func(in []reflect.Value) []reflect.Value {
		args := make([]Value, len(in))
		for i, rv := range in {
			args[i] = Value{rv: rv}
		}
		results := fb.impl(args)
		out := make([]reflect.Value, len(results))
		for i, v := range results {
			out[i] = v.rv
		}
		return out
	}

	rv := reflect.MakeFunc(funcType, wrapper)
	return Func{rv: rv, rt: funcType}
}


// ============================================================================
// Additional Helpers
// ============================================================================

// New creates a pointer to a new zero value of type T.
func New[T any]() Ptr {
	var zero T
	elemType := reflect.TypeOf(zero)
	if elemType == nil {
		elemType = AnyType
	}
	rv := reflect.New(elemType)
	return Ptr{rv: rv, elemType: elemType}
}

// NewOf creates a pointer to a new zero value of the given type.
func NewOf(t Type) Ptr {
	rv := reflect.New(t)
	return Ptr{rv: rv, elemType: t}
}

// Zero creates a zero value of type T.
func Zero[T any]() Value {
	var zero T
	return Value{rv: reflect.ValueOf(zero)}
}

// ZeroOf creates a zero value of the given type.
func ZeroOf(t Type) Value {
	return Value{rv: reflect.Zero(t)}
}

// TypeOf returns the Type of type T.
func TypeOf[T any]() Type {
	var zero T
	t := reflect.TypeOf(zero)
	if t == nil {
		return reflect.TypeOf((*T)(nil)).Elem()
	}
	return t
}

// ============================================================================
// Type Construction Helpers
// ============================================================================

// PtrTo returns a pointer type to the given type.
func PtrTo(t Type) Type {
	return reflect.PointerTo(t)
}

// SliceTypeOf returns a slice type with the given element type.
func SliceTypeOf(elemType Type) Type {
	return reflect.SliceOf(elemType)
}

// MapTypeOf returns a map type with the given key and value types.
func MapTypeOf(keyType, valType Type) Type {
	return reflect.MapOf(keyType, valType)
}

// ChanTypeOf returns a channel type with the given element type and direction.
func ChanTypeOf(elemType Type, dir ChanDir) Type {
	return reflect.ChanOf(dir, elemType)
}

// ArrayTypeOf returns an array type with the given length and element type.
func ArrayTypeOf(length int, elemType Type) Type {
	return reflect.ArrayOf(length, elemType)
}

// FuncTypeOf returns a function type with the given signature.
func FuncTypeOf(in []Type, out []Type, variadic bool) Type {
	return reflect.FuncOf(in, out, variadic)
}
