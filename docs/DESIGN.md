# gref Design Document

## Overview

`gref` is a hierarchical abstraction over Go's `reflect` package that provides a type-safe, fluent API for runtime reflection operations. It prioritizes ergonomics, discoverability, and clear error semantics while maintaining full access to reflection capabilities.

## Design Goals

1. **Hierarchical Navigation**: Chain through values naturally: `From(v).Struct().Field("Name").Tag("json")`
2. **Type Safety**: Use Go generics for type-safe extraction: `Get[string](field)`
3. **Clear Error Semantics**: Distinguish programmer errors (panic) from runtime conditions (error)
4. **Discoverability**: Each type has only relevant methods; IDE autocomplete guides usage
5. **Minimal Ceremony**: Common operations should be concise
6. **No Magic**: Explicit over implicit; users understand what's happening
7. **Orthogonality**: No redundant functions; each capability has exactly one way to access it

## Orthogonality Principles

The API avoids redundancy. Each operation has exactly one home:

| Need | Solution | NOT |
|------|----------|-----|
| Create slice (compile-time) | `Make[[]T]().Slice(len, cap)` | ~~`MakeSlice[T]()`~~ |
| Create map (compile-time) | `Make[map[K]V]().Map()` | ~~`MakeMap[K,V]()`~~ |
| Create chan (compile-time) | `Make[chan T]().Chan(buf)` | ~~`MakeChan[T]()`~~ |
| Create struct (compile-time) | `Make[T]().Struct()` | ~~`MakeStruct[T]()`~~ |
| Create func (compile-time) | `Make[F]().Impl(fn)` | ~~`MakeFunc[F]().Impl()`~~ |
| Create slice (runtime) | `Def().Slice(elemType, len, cap)` | ~~`SliceDef()`~~ |
| Create map (runtime) | `Def().Map(keyType, valType)` | ~~`MapDef()`~~ |
| Create chan (runtime) | `Def().Chan(elemType, buf)` | ~~`ChanDef()`~~ |
| Create struct (runtime) | `Def().Struct(name).Field(...).Struct()` | ~~`StructDef().Build()`~~ |
| Create func (runtime) | `Def().Func(name).Arg(...).Impl(...)` | ~~`FuncDef().Build()`~~ |
| Iterate fields | `s.Fields().Iter()` with range | ~~`.Each()` callback only~~ |
| Iterate slice | `sl.Iter()` with range | ~~`.Each()` callback only~~ |
| Iterate map | `m.Iter()` with range | ~~`.Each()` callback only~~ |
| Get with default | `TryGet[T](v).Or(def)` | ~~`GetOr[T](v, def)`~~ |
| Optional field | `TryField(name).Or(def)` | ~~`(Field, bool)`~~ |
| Optional index | `TryIndex(i).Or(def)` | ~~`(Element, bool)`~~ |
| Optional map get | `TryGet(key).Or(def)` | ~~`(Entry, bool)`~~ |
| Check type compat | `Is[T](v)` | ~~`CanGet[T](v)`~~ |
| Exact type check | `IsExactly[T](v)` | ~~`IsType[T](v)`~~ |
| Get string | `Get[string](v)` | ~~`GetString(v)`~~ |
| Check if nil | `From(v).IsNil()` | ~~`gref.IsNil(v)`~~ |
| Deep equality | `reflect.DeepEqual(a, b)` | ~~`gref.DeepEqual(a, b)`~~ |

**Rationale**: 
- Unified `Make[T]()` reduces 5 compile-time creation functions to 1
- Unified `Def()` reduces 5 runtime creation functions to 1
- Args on narrowing methods (`.Slice(len, cap)`) keep entry points simple
- No `.Build()` needed - terminal methods return usable values directly
- `.Iter()` enables idiomatic Go range loops (Go 1.23+)
- `Result[T]` and `Option[T]` provide chainable APIs consistent across all Try* operations
- Generics (`Get[T]`) eliminate type-specific extractors
- Methods on types are preferred over package functions for discoverability

## Architecture

### Type Hierarchy

```
From(v) → Value
            │
            ├── .Struct()  → Struct  ──→ .Field()   → Field
            │                        ──→ .Method()  → Func
            │
            ├── .Slice()   → Slice   ──→ .Index()   → Element
            │                        ──→ .First()   → Element
            │                        ──→ .Last()    → Element
            │
            ├── .Map()     → Map     ──→ .Get()     → Entry
            │
            ├── .Chan()    → Chan
            │
            ├── .Func()    → Func    ──→ .Call()    → Results
            │
            ├── .Ptr()     → Ptr     ──→ .Elem()    → Value
            │
            └── .Iface()   → Iface   ──→ .Underlying() → Value
```

### Core Interface

```go
// Valuable is implemented by all types that wrap a reflect.Value
type Valuable interface {
    reflectValue() reflect.Value
}
```

All specialized types (`Value`, `Struct`, `Field`, `Slice`, `Element`, `Map`, `Entry`, `Chan`, `Func`, `Ptr`, `Iface`) implement `Valuable`, allowing uniform extraction via `Get[T]()`.

## API Conventions

### Entry Point

```go
v := gref.From(anyValue)  // Returns Value
```

`From()` is the single entry point. It accepts any Go value and returns a `Value` that can be navigated to specialized types.

### Navigation Methods

Navigation methods transition from one type to a more specialized type:

| Method | From | To | Behavior |
|--------|------|-----|----------|
| `.Struct()` | Value, Field, Element, Entry, Ptr, Iface | Struct | Auto-derefs pointers |
| `.Slice()` | Value, Field, Element, Entry, Ptr, Iface | Slice | Auto-derefs pointers |
| `.Map()` | Value, Field, Element, Entry, Ptr, Iface | Map | Auto-derefs pointers |
| `.Chan()` | Value, Field, Element, Entry, Ptr, Iface | Chan | Auto-derefs pointers |
| `.Func()` | Value, Field, Element, Entry, Ptr, Iface | Func | Auto-derefs pointers |
| `.Ptr()` | Value, Field, Element, Entry | Ptr | NO auto-deref |
| `.Iface()` | Value, Field, Element, Entry, Ptr | Iface | For interface inspection |

**Auto-dereferencing**: Most navigation methods automatically dereference pointers to reach the underlying value. This matches common usage patterns where `*User` and `User` are treated similarly.

**Exception**: `.Ptr()` does NOT auto-dereference because its purpose is explicit pointer manipulation.

### Value Creation Conventions

Two distinct conventions based on when types are known:

#### Compile-time: `Make[T]()`

The unified `Make[T]()` function creates refl wrappers for compile-time known types.
Configuration arguments are passed to the narrowing methods:

```go
gref.Make[[]int]().Slice(len, cap)      // Slice with length and capacity
gref.Make[map[string]int]().Map()       // map[string]int  
gref.Make[chan string]().Chan(bufSize)  // chan string with buffer
gref.Make[User]().Struct()              // Settable User struct
gref.Make[func(string) bool]().Impl(fn) // Function with implementation
```

The `Maker[T]` type returned by `Make[T]()` provides:
- `.Slice(len, cap)` - create slice (args optional, default 0)
- `.Chan(bufSize)` - create channel (arg optional, default 0)  
- `.Map()`, `.Struct()`, `.Ptr()` - create map, struct, pointer
- `.Impl(fn)`, `.ImplDynamic(fn)` - function implementation
- `.Type()` - get reflect.Type

#### Runtime: `Def()`

When types are determined at runtime, use the unified `Def()` function:

```go
gref.Def().Slice(elemType, len, cap)           // Returns Slice directly
gref.Def().Map(keyType, valType)               // Returns Map directly
gref.Def().Chan(elemType, bufSize)             // Returns Chan directly
gref.Def().Struct("Name").Field(...).Struct()  // .Struct() returns Struct
gref.Def().Func("name").Arg(...).Return(...).Impl(...)  // .Impl() returns Func
```

No `.Build()` method needed - values are ready to use after the terminal method.

### Iteration

All collection types support Go 1.23+ range-over-func via `.Iter()`:

```go
// Struct fields
for field := range s.Fields().Iter() { ... }
for field := range s.Fields().Exported().Iter() { ... }
for field := range s.Fields().WithTag("json").Iter() { ... }

// Slice elements
for elem := range sl.Iter() { ... }

// Map entries
for entry := range m.Iter() { ... }
```

The `.Each()` callback style is also available for all collection types.

### Extraction Functions

Type-safe value extraction with multiple styles:

```go
val, err := gref.Get[T](v)              // Returns error on mismatch
val := gref.MustGet[T](v)               // Panics on mismatch
val := gref.TryGet[T](v).Or(default)    // Chainable with default
val := gref.TryGet[T](v).OrZero()       // Zero value on failure
val, ok := gref.TryGet[T](v).Value()    // Traditional (val, ok) pattern
```

### Type Checking

```go
gref.Is[int](v)              // Can extract as int? (includes conversions)
gref.IsExactly[int64](v)     // Exact type match only?
gref.IsKind(v, reflect.Struct) // Kind check
```

### Result and Option Types

The API uses two wrapper types for fallible operations:

| Type | Semantics | Used By |
|------|-----------|---------|
| `Result[T]` | Success/Failure (type extraction can fail) | `TryGet[T]()` |
| `Option[T]` | Present/Absent (data may not exist) | `TryField()`, `TryIndex()`, `TryGet()` (map), `TryElem()`, `TryMethod()` |

Both types share a common API:

```go
.Or(default)     // Value or default
.OrZero()        // Value or zero value  
.OrElse(fn)      // Value or lazy default
.Value()         // (T, bool) - traditional Go pattern
.Must()          // Value or panic
```

Additional methods:
- `Result[T]`: `.Ok()` - returns bool
- `Option[T]`: `.Some()`, `.None()` - returns bool

**Rationale**: These types provide chainable, expressive handling of fallible operations while maintaining compatibility with traditional Go patterns via `.Value()`.

## Error Handling Philosophy

### Panics: Programmer Errors

Panics indicate bugs in the calling code—mistakes that should be caught during development:

| Operation | Panic Condition |
|-----------|-----------------|
| `From(v).Struct()` | v is not a struct (or pointer to struct) |
| `From(v).Slice()` | v is not a slice or array |
| `s.Field("Name")` | Field "Name" doesn't exist |
| `sl.Index(5)` | Index 5 is out of bounds |
| `m.Get("key")` | Key doesn't exist in map |
| `MakeStruct[int]()` | int is not a struct type |

**Rationale**: These are static properties of the code. If you call `.Struct()` on something that isn't a struct, your code is wrong regardless of runtime conditions.

### Errors: Runtime Conditions

Errors indicate conditions that may legitimately vary at runtime:

| Operation | Error Condition |
|-----------|-----------------|
| `Get[int](stringField)` | Type mismatch during extraction |
| `Get[T](nilValue)` | Value is nil/invalid |

**Rationale**: Type mismatches during extraction can occur when processing dynamic data (JSON, user input, etc.) and should be handleable.

### Try* Variants: Graceful Handling

For cases where panics are undesirable, `Try*` methods return `Option[T]` for chainable access:

```go
// Using Or() for defaults
field := s.TryField("MaybeExists").Or(defaultField)
elem := sl.TryIndex(i).OrZero()
entry := m.TryGet(key).OrElse(createDefault)
val := p.TryElem().Or(defaultValue)
method := s.TryMethod("String").Or(defaultMethod)

// Using Value() for traditional Go pattern
if field, ok := s.TryField("Name").Value(); ok { ... }
if elem, ok := sl.TryIndex(i).Value(); ok { ... }
if entry, ok := m.TryGet(key).Value(); ok { ... }

// Checking presence
if s.TryField("Name").Some() { ... }
if m.TryGet("key").None() { ... }
```

## Specialized Types

### Value

The root type returned by `From()`. Provides:
- Kind/Type inspection
- Navigation to specialized types
- Convenience extractors (`.String()`, `.Int()`, etc.)

### Struct

Represents struct values. Provides:
- Field access by name (with dot notation for nested: `"Address.City"`)
- Field iteration with filtering (`.Fields().Exported().WithTag("json")`)
- Method access
- Conversion to map

### Field

Represents a struct field with both value and metadata:
- Value operations (Get, Set, navigation)
- Metadata (Name, Tag, Index, IsExported, IsEmbedded)
- Tag parsing (`ParsedTag("json")` returns structured tag info)

### Slice

Represents slices and arrays. Provides:
- Element access (Index, First, Last)
- Modification (Set, Append, SubSlice, Clear)
- Functional operations (Map, Filter, Reduce, Find, All, Any)
- Iteration (Each)

### Element

Represents a slice/array element with position:
- Value operations
- Position metadata
- Navigation to nested types

### Map

Represents maps. Provides:
- Key-value access (Get, Set, Delete)
- Bulk operations (Keys, Values, Entries)
- Functional operations (Filter, MapValues)
- Merge, Clone

### Entry

Represents a map entry with key and value:
- Key access (both as Value and as interface{})
- Value operations
- Navigation to nested types

### Chan

Represents channels. Provides:
- Send/Receive (blocking and non-blocking)
- Direction checking
- Range iteration
- Close

### Func

Represents functions. Provides:
- Signature inspection
- Call with automatic argument conversion
- Results handling with error extraction
- Partial application (Bind)

### Results

Represents function call results:
- Indexed access
- Error extraction (if last return is error)
- Unpacking into variables

### Ptr

Represents pointers explicitly (no auto-deref):
- Indirection depth calculation
- Multi-level dereference (DerefAll, DerefN)
- Allocation (Alloc, AllocIfNil)
- Nil handling

### Iface

Represents interface values:
- Nil checks (including typed nil detection)
- Concrete type inspection
- Underlying value extraction
- Type assertion checking

## Predefined Types

For runtime type construction, commonly-used types are exported:

```go
gref.StringType     // reflect.TypeOf("")
gref.IntType        // reflect.TypeOf(0)
gref.Int64Type      // reflect.TypeOf(int64(0))
gref.Float64Type    // reflect.TypeOf(float64(0))
gref.BoolType       // reflect.TypeOf(false)
gref.ByteType       // reflect.TypeOf(uint8(0))
gref.AnyType        // interface{} type
gref.ErrorType      // error interface type
// ... etc
```

## Comparison with stdlib reflect

| Task | stdlib reflect | refl |
|------|---------------|------|
| Get field value | `v.FieldByName("X").Interface()` | `Get[T](s.Field("X"))` |
| Set field | `v.FieldByName("X").Set(reflect.ValueOf(x))` | `s.SetField("X", x)` |
| Get nested field | `v.FieldByName("A").FieldByName("B")` | `s.Field("A.B")` |
| Check field exists | `_, ok := t.FieldByName("X")` | `s.TryField("X").Some()` |
| Get field or default | Manual check + fallback | `s.TryField("X").Or(def)` |
| Get tag | `t.Field(i).Tag.Get("json")` | `s.Field("X").Tag("json")` |
| Iterate fields | `for i := 0; i < t.NumField(); i++` | `for f := range s.Fields().Iter()` |
| Iterate slice | `for i := 0; i < v.Len(); i++` | `for e := range sl.Iter()` |
| Iterate map | `iter := v.MapRange(); for iter.Next()` | `for e := range m.Iter()` |
| Filter slice | Manual loop | `sl.Filter(predicate)` |
| Call function | `v.Call([]reflect.Value{...})` | `f.Call(arg1, arg2)` |
| Create map | `reflect.MakeMap(reflect.MapOf(...))` | `Make[map[K]V]().Map()` |
| Create slice | `reflect.MakeSlice(...)` | `Make[[]T]().Slice(len, cap)` |
| Get with default | Manual check + fallback | `TryGet[T](v).Or(def)` |

## Design Decisions

### Why Panic on Kind Mismatch?

Alternative considered: Return `(Struct, error)` from `.Struct()`.

Rejected because:
1. Kind is a static property—if code calls `.Struct()`, it expects a struct
2. Error returns would require checking at every navigation step
3. `Try*` variants exist for cases needing graceful handling
4. Matches Go idiom (e.g., type assertions panic without `,ok`)

### Why Error on Type Extraction?

Alternative considered: Panic on `Get[int](stringField)`.

Rejected because:
1. Extraction often happens on dynamic data
2. Users may legitimately not know the exact type
3. Multiple extraction attempts may be needed (`Get[int]` then `Get[string]`)
4. `MustGet[T]` exists for when panic is desired

### Why Auto-Deref in Navigation?

Alternative considered: Require explicit `.Ptr().Elem()` always.

Accepted because:
1. Most code treats `*User` and `User` identically for field access
2. Explicit pointer handling via `.Ptr()` when needed
3. Matches intuition from standard Go (method receivers, JSON, etc.)

### Why Separate Ptr Type?

Alternative considered: Auto-deref everywhere, no Ptr type.

Rejected because:
1. Sometimes pointer manipulation is the goal
2. Need to distinguish `*T` from `T` for some operations
3. Multi-level pointers (`**T`) require explicit handling
4. Allocation operations need pointer context

### Why Specialized Types vs Methods on Value?

Alternative considered: All operations on Value with kind checking.

Rejected because:
1. Too many methods on one type (poor discoverability)
2. IDE autocomplete becomes useless
3. Easy to call wrong method for kind
4. Specialized types document valid operations

## Performance Considerations

`gref` adds a thin wrapper over `reflect`. Performance characteristics:

1. **Allocation**: Each specialized type is a small struct (usually 2-3 fields)
2. **Method calls**: Direct, no virtual dispatch
3. **Caching**: No automatic caching; users can cache Type if needed
4. **Hot paths**: Consider stdlib reflect for performance-critical code

For most use cases (configuration, serialization, testing), the ergonomic benefits outweigh the minimal overhead.

## Future Considerations

Potential additions (not currently implemented):

1. **Struct field caching**: Cache field lookups by type
2. **Concurrent map operations**: Thread-safe Map wrapper
3. **Validation helpers**: Built-in struct validation using tags
4. **Codec support**: JSON/YAML encoding helpers
5. **Diff/Patch**: Struct comparison and modification

## Summary

`gref` provides a hierarchical, type-safe reflection API that:

- Uses `From()` as single entry point
- Returns specialized types with relevant methods
- Panics on programmer errors, returns errors on runtime conditions
- Offers `Make[T]()` for compile-time and `Def()` for runtime type construction
- Auto-dereferences pointers in navigation (except `.Ptr()`)
- Provides `Get[T]()` family for type-safe extraction

The goal is making reflection code readable, maintainable, and hard to misuse.
