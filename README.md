# gref - Hierarchical Reflection for Go

A type-safe, hierarchical abstraction over Go's `reflect` package.

## Design Principles

- **Hierarchical API**: `From(v).Struct().Field("Name").Tag("json")`
- **Type-safe extraction**: `Get[T](value)` returns `(T, error)`
- **Panic on kind mismatch**: `.Struct()` panics if value isn't a struct
- **Error on extraction mismatch**: `Get[int](stringField)` returns error
- **Specialized types**: Each navigation step returns a type with relevant methods
- **Auto-dereferencing**: Navigation methods transparently handle pointers

## Installation

```bash
go get github.com/vladimirvivien/gref
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/vladimirvivien/gref"
)

type Address struct {
    City   string `json:"city"`
    Street string `json:"street"`
}

type User struct {
    Name    string   `json:"name" validate:"required"`
    Age     int      `json:"age"`
    Address *Address `json:"address"`
    Tags    []string `json:"tags"`
}

func main() {
    user := &User{
        Name: "Alice",
        Age:  30,
        Address: &Address{City: "Seattle", Street: "123 Main St"},
        Tags: []string{"developer", "gopher"},
    }

    // =========================================
    // Basic field access
    // =========================================
    
    name, err := gref.Get[string](gref.From(user).Struct().Field("Name"))
    if err != nil {
        panic(err)
    }
    fmt.Println("Name:", name) // Name: Alice

    // =========================================
    // Field metadata (tags)
    // =========================================
    
    tag := gref.From(user).Struct().Field("Name").Tag("validate")
    fmt.Println("Validate tag:", tag) // Validate tag: required

    // =========================================
    // Nested field access (dot notation)
    // =========================================
    
    city, _ := gref.Get[string](gref.From(user).Struct().Field("Address.City"))
    fmt.Println("City:", city) // City: Seattle

    // =========================================
    // Explicit navigation (equivalent to dot notation)
    // =========================================
    
    city2, _ := gref.Get[string](
        gref.From(user).Struct().Field("Address").Struct().Field("City"),
    )
    fmt.Println("City (explicit):", city2) // City (explicit): Seattle

    // =========================================
    // Slice navigation
    // =========================================
    
    firstTag, _ := gref.Get[string](gref.From(user).Struct().Field("Tags").Slice().First())
    fmt.Println("First tag:", firstTag) // First tag: developer
}
```

## Core Concepts

### The Navigation Chain

Every reflection operation starts with `From(v)` which returns a `Value`. From there, you navigate to specialized types:

```
From(v) → Value
    .Struct() → Struct
    .Slice()  → Slice
    .Map()    → Map
    .Chan()   → Chan
    .Func()   → Func
    .Ptr()    → Ptr
    .Iface()  → Iface
```

Each specialized type has methods appropriate for that kind:

```go
// Struct has Field(), Fields(), ToMap(), Method(), etc.
gref.From(user).Struct().Field("Name")

// Slice has Index(), First(), Last(), Filter(), Map(), etc.
gref.From(slice).Slice().Index(0)

// Map has Get(), Set(), Keys(), Values(), etc.
gref.From(m).Map().Get("key")
```

### Type Safety with Get[T]

Extract typed values using `Get[T]`:

```go
// Returns (string, error) - error if types don't match
name, err := gref.Get[string](field)

// Panics if types don't match
name := gref.MustGet[string](field)

// Chainable with default
name := gref.TryGet[string](field).Or("anonymous")

// Traditional Go-idiomatic (value, ok) style
if name, ok := gref.TryGet[string](field).Value(); ok {
    fmt.Println(name)
}
```

### Panic vs Error

**Panics** occur for programmer errors (wrong kind):
```go
gref.From(42).Struct()  // PANIC: int is not a struct
```

**Errors** occur for runtime conditions (type mismatch in extraction):
```go
val, err := gref.Get[int](stringField)  // err != nil
```

## API Reference

### Value

Entry point for all reflection operations.

```go
// Create from any value
v := gref.From(myValue)

// Basic info
v.Kind()      // reflect.Kind
v.Type()      // reflect.Type
v.IsValid()   // bool
v.IsNil()     // bool
v.IsZero()    // bool
v.Interface() // any

// Navigate to specialized types (panics on wrong kind)
v.Struct()    // Struct
v.Slice()     // Slice
v.Map()       // Map
v.Chan()      // Chan
v.Func()      // Func
v.Ptr()       // Ptr (does NOT auto-deref)
v.Iface()     // Iface

// Convenience extractors (panic on wrong kind)
v.String()    // string
v.Int()       // int64
v.Uint()      // uint64
v.Float()     // float64
v.Bool()      // bool
v.Bytes()     // []byte
v.Len()       // int
v.Cap()       // int
```

### Struct

Operations on struct values.

```go
s := gref.From(myStruct).Struct()

// Field access
f := s.Field("Name")                      // Field (panics if not found)
f := s.Field("Address.City")              // Nested with dot notation
f := s.TryField("Name").Or(defaultField)  // Option[Field] - with default
f := s.TryField("Name").OrZero()          // Option[Field] - zero if missing
if f, ok := s.TryField("Name").Value(); ok { ... }  // Traditional pattern

// Field info
s.NumFields()                  // int

// Field iteration with range (Go 1.23+)
for field := range s.Fields().Iter() {
    fmt.Println(field.Name(), field.Tag("json"))
}

// Field iteration with callback
s.Fields().Each(func(f gref.Field) bool {
    fmt.Println(f.Name(), f.Tag("json"))
    return true // continue
})

// Filtered iteration
for field := range s.Fields().Exported().Iter() { ... }
for field := range s.Fields().WithTag("json").Iter() { ... }
s.Fields().Exported().Each(...)
s.Fields().WithTag("json").Each(...)

// Methods
f := s.Method("DoSomething")              // Func (panics if not found)
f := s.TryMethod("X").Or(defaultMethod)   // Option[Func]
s.NumMethods()                            // int

// Modification
s.SetField("Name", "Bob")

// Conversion
m := s.ToMap("json")           // map[string]any

// Interface check
s.Implements((*fmt.Stringer)(nil))
```

### Field

Represents a struct field with value and metadata.

```go
f := gref.From(user).Struct().Field("Name")

// Metadata
f.Name()                       // string
f.Tag("json")                  // string
f.TagLookup("json")            // string, bool
f.RawTag()                     // reflect.StructTag
f.ParsedTag("json")            // ParsedTag{Name, Options, HasOption()}
f.Type()                       // reflect.Type
f.Index()                      // []int
f.IsExported()                 // bool
f.IsEmbedded()                 // bool

// Value operations
f.Interface()                  // any
f.IsNil()                      // bool
f.IsZero()                     // bool
f.CanSet()                     // bool
f.Set(newValue)                // (panics if not settable)
f.Kind()                       // reflect.Kind

// Navigate into field value
f.Struct()                     // if field is a struct
f.Slice()                      // if field is a slice
f.Map()                        // if field is a map
f.Ptr()                        // if field is a pointer
```

### Slice

Operations on slices and arrays.

```go
sl := gref.From(mySlice).Slice()

// Info
sl.Len()                       // int
sl.Cap()                       // int
sl.IsEmpty()                   // bool
sl.IsNil()                     // bool
sl.IsArray()                   // bool
sl.ElemType()                  // reflect.Type

// Element access
e := sl.Index(0)                       // Element (panics if out of bounds)
e := sl.TryIndex(0).Or(defaultElem)    // Option[Element] - with default
e := sl.TryIndex(0).OrZero()           // Option[Element] - zero if OOB
if e, ok := sl.TryIndex(0).Value(); ok { ... }  // Traditional pattern
e := sl.First()                        // Element
e := sl.Last()                         // Element

// Modification (returns new Slice)
sl = sl.Set(0, value)
sl = sl.Append(1, 2, 3)
sl = sl.AppendSlice(other)
sl = sl.SubSlice(1, 4)
sl = sl.Clear()

// Iteration with range (Go 1.23+)
for elem := range sl.Iter() {
    val, _ := gref.Get[int](elem)
    fmt.Println(elem.Position(), val)
}

// Iteration with callback
sl.Each(func(e gref.Element) bool {
    return true
})

// Functional operations
filtered := sl.Filter(func(e gref.Element) bool {
    val, _ := gref.Get[int](e)
    return val > 10
})

mapped := sl.Map(func(e gref.Element) any {
    val, _ := gref.Get[int](e)
    return val * 2
})

sum := sl.Reduce(0, func(acc any, e gref.Element) any {
    val, _ := gref.Get[int](e)
    return acc.(int) + val
})

// Search
e, ok := sl.Find(predicate)
idx := sl.FindIndex(predicate)
sl.Contains(value)
sl.All(predicate)
sl.Any(predicate)

// Transform
reversed := sl.Reverse()
```

### Element

Represents a slice/array element with position.

```go
e := gref.From(slice).Slice().Index(0)

// Metadata
e.Position()                   // int (index in slice)
e.Type()                       // reflect.Type

// Value operations
e.Interface()                  // any
e.IsNil()                      // bool
e.IsZero()                     // bool
e.Set(newValue)

// Navigation (if element is composite)
e.Struct()                     // Struct
e.Slice()                      // Slice
e.Map()                        // Map
```

### Map

Operations on maps.

```go
m := gref.From(myMap).Map()

// Info
m.Len()                        // int
m.IsEmpty()                    // bool
m.IsNil()                      // bool
m.KeyType()                    // reflect.Type
m.ValType()                    // reflect.Type

// Access
e := m.Get("key")                       // Entry (panics if not found)
e := m.TryGet("key").Or(defaultEntry)   // Option[Entry] - with default
e := m.TryGet("key").OrZero()           // Option[Entry] - zero if missing
if e, ok := m.TryGet("key").Value(); ok { ... }  // Traditional pattern
m.Has("key")                            // bool

// Modification
m = m.Set("key", value)
m = m.Delete("key")
m = m.Clear()

// Bulk
keys := m.Keys()               // Slice
values := m.Values()           // Slice
entries := m.Entries()         // []Entry

// Iteration with range (Go 1.23+)
for entry := range m.Iter() {
    key, _ := gref.Get[string](entry.Key())
    val, _ := gref.Get[int](entry)
    fmt.Println(key, val)
}

// Iteration with callback
m.Each(func(e gref.Entry) bool {
    fmt.Println(e.KeyInterface(), e.Interface())
    return true
})

// Functional
filtered := m.Filter(func(e gref.Entry) bool { ... })
mapped := m.MapValues(func(e gref.Entry) any { ... })

// Operations
merged := m.Merge(other)
cloned := m.Clone()

// Conversion
goMap := m.ToGoMap()           // map[string]any (key must be string)
```

### Entry

Represents a map entry with key and value.

```go
e := gref.From(m).Map().Get("key")

// Key
e.Key()                        // Value
e.KeyInterface()               // any
e.KeyType()                    // reflect.Type

// Value
e.Interface()                  // any
e.IsNil()                      // bool
e.IsZero()                     // bool
e.ValType()                    // reflect.Type

// Navigation (if value is composite)
e.Struct()                     // Struct
e.Slice()                      // Slice
e.Map()                        // Map
```

### Chan

Operations on channels.

```go
c := gref.From(myChan).Chan()

// Info
c.Len()                        // int (buffered count)
c.Cap()                        // int
c.IsNil()                      // bool
c.IsFull()                     // bool
c.IsEmpty()                    // bool
c.Dir()                        // ChanDir
c.CanSend()                    // bool
c.CanRecv()                    // bool

// Send
c.Send(value)                  // Chan (blocking)
c.TrySend(value)               // bool (non-blocking)
c.SendAll(1, 2, 3)             // Chan

// Receive
val, open := c.Recv()          // Value, bool (blocking)
val, received, open := c.TryRecv() // non-blocking

// Close
c.Close()

// Iteration
c.Range(func(v gref.Value) bool {
    return true
})

// Drain buffered values
values := c.Drain()            // []Value
```

### Func

Operations on functions.

```go
f := gref.From(myFunc).Func()

// Signature
f.NumIn()                      // int
f.NumOut()                     // int
f.In(0)                        // reflect.Type
f.Out(0)                       // reflect.Type
f.IsVariadic()                 // bool
f.IsNil()                      // bool
f.ReturnsError()               // bool
f.Signature()                  // Signature

// Call
results := f.Call(arg1, arg2)
results.Len()                  // int
results.First()                // Value
results.Last()                 // Value
results.Index(0)               // Value
results.Error()                // error (if last return is error)
results.All()                  // []Value
results.Collect()              // []any
results.Unpack(&a, &b)         // error

// Partial application
bound := f.Bind("hello")       // Func with first arg pre-filled
```

### Ptr

Explicit pointer operations (no auto-deref).

```go
p := gref.From(&x).Ptr()

// Info
p.IsNil()                      // bool
p.ElemType()                   // reflect.Type
p.IndirectionDepth()           // int (**int → 2)
p.UltimateType()               // final non-pointer type

// Dereference
v := p.Elem()                  // Value (panics if nil)
v, ok := p.TryElem()           // Value, bool
v := p.DerefAll()              // dereference all levels
v := p.DerefN(2)               // dereference N levels

// Modify
p.Set(newValue)                // set pointed-to value
p.SetNil()                     // set pointer to nil

// Allocate
p.Alloc()                      // allocate new value
p.AllocIfNil()                 // allocate only if nil

// Navigation
p.Struct()                     // Struct (deref then get struct)
p.Slice()                      // Slice
p.Map()                        // Map
```

### Iface

Operations on interface values.

```go
i := gref.From(&myInterface).Ptr().Elem().Iface()

// Nil checks
i.IsNil()                      // bool
i.HasTypedNil()                // bool (interface not nil, but value is)

// Type info
i.ConcreteType()               // reflect.Type
i.ConcreteKind()               // reflect.Kind

// Underlying value
v := i.Underlying()            // Value (panics if nil)
v, ok := i.TryUnderlying()     // Value, bool

// Type checks
i.CanAssertTo(targetType)      // bool
i.Implements((*fmt.Stringer)(nil)) // bool

// Navigation
i.Struct()                     // Struct
i.Slice()                      // Slice
i.Map()                        // Map
i.Func()                       // Func

// Methods
f := i.Method("String")        // Func
i.NumMethods()                 // int
```

## Creating Values Dynamically

### Compile-time Known Types: Make[T]()

The unified `Make[T]()` function creates refl wrappers for any Go type:

```go
// Slice - pass (length, capacity) to Slice()
sl := gref.Make[[]int]().Slice(0, 10).Append(1, 2, 3)

// Map - no args needed
m := gref.Make[map[string]int]().Map().Set("one", 1)

// Channel - pass bufferSize to Chan()
c := gref.Make[chan string]().Chan(10).Send("hello")

// Struct - no args needed
user := gref.Make[User]().Struct()
user.SetField("Name", "Alice")

// Function - use .Impl() for implementation
f := gref.Make[func(string, int) bool]().Impl(
    func(name string, age int) bool {
        return age >= 18
    },
)

// Pointer - no args needed
p := gref.Make[*User]().Ptr()
```

The `Maker[T]` returned by `Make[T]()` provides:
- `.Slice(len, cap)` - create slice with length and capacity (both optional)
- `.Chan(bufSize)` - create channel with buffer size (optional, default 0)
- `.Map()`, `.Struct()`, `.Ptr()` - create map, struct, or pointer
- `.Impl(fn)` - provide function implementation (for func types)
- `.Type()` - get reflect.Type

### Helper Functions

```go
// Pointer to new zero value
p := gref.New[Person]()
s := p.Struct()

// Zero value wrapped in Value
v := gref.Zero[int]()
```

### Runtime-determined Types: Def()

When types are only known at runtime, use the unified `Def()` function:

```go
// Slice with runtime element type
sl := gref.Def().Slice(gref.IntType, 0, 10)
sl = sl.Append(1, 2, 3)

// Map with runtime key/value types
m := gref.Def().Map(gref.StringType, gref.IntType)
m = m.Set("count", 42)

// Channel with runtime element type
c := gref.Def().Chan(gref.StringType, 10)
c.Send("hello")

// Struct with runtime fields - Struct() returns the value directly
s := gref.Def().Struct("Person").
    Field(gref.FieldDef{Name: "Name", Type: gref.StringType, Tag: `json:"name"`}).
    Field(gref.FieldDef{Name: "Age", Type: gref.IntType}).
    Set("Name", "Alice").
    Set("Age", 30).
    Struct()

// Function with runtime signature - Impl() returns the func directly
fn := gref.Def().Func("isAdult").
    Arg(gref.ArgDef{Name: "name", Type: gref.StringType}).
    Arg(gref.ArgDef{Name: "age", Type: gref.IntType}).
    Return(gref.ReturnDef{Type: gref.BoolType}).
    Impl(func(args []gref.Value) []gref.Value {
        age, _ := gref.Get[int](args[1])
        return []gref.Value{gref.From(age >= 18)}
    })
result := fn.Call("Alice", 25)

// Copy and modify existing struct
user := gref.Make[User]().Struct()
copy := gref.StructFrom(user).Set("Name", "Bob").Struct()
```

### Predefined Types

```go
gref.StringType    // reflect.TypeOf("")
gref.IntType       // reflect.TypeOf(0)
gref.Int64Type     // reflect.TypeOf(int64(0))
gref.Float64Type   // reflect.TypeOf(float64(0))
gref.BoolType      // reflect.TypeOf(false)
gref.ByteType      // reflect.TypeOf(uint8(0))
gref.AnyType       // interface{} type
gref.ErrorType     // error interface type
// ... and more
```

### Type Construction

```go
// Get reflect.Type for a Go type
intType := gref.TypeOf[int]()
stringType := gref.TypeOf[string]()

// Construct composite types
ptrType := gref.PtrTo(intType)                    // *int
sliceType := gref.SliceTypeOf(intType)            // []int
mapType := gref.MapTypeOf(stringType, intType)    // map[string]int
chanType := gref.ChanTypeOf(intType, gref.BothDir) // chan int
arrayType := gref.ArrayTypeOf(10, intType)        // [10]int
```

## Extraction Functions

```go
// Get with error handling
val, err := gref.Get[string](field)

// Must - panics on error
val := gref.MustGet[string](field)

// TryGet - returns Result[T] for chainable access
val := gref.TryGet[string](field).Or("default")      // with default
val := gref.TryGet[int](field).OrZero()              // zero value on failure
val := gref.TryGet[Config](field).OrElse(loadConfig) // lazy default
ok := gref.TryGet[string](field).Ok()                // just check success
val, ok := gref.TryGet[string](field).Value()        // traditional (val, ok)

// Type checking
gref.Is[int](v)              // can extract as int? (includes conversions)
gref.IsExactly[int64](v)     // exact type match only?
gref.IsKind(v, reflect.Struct) // kind check
```

## Option Type for Try* Methods

All `Try*` methods return `Option[T]` for consistent, chainable handling of missing data:

```go
// Struct field access
field := s.TryField("MaybeExists").Or(defaultField)
field := s.TryField("Name").OrZero()
if field, ok := s.TryField("Name").Value(); ok { ... }

// Slice index access  
elem := sl.TryIndex(i).Or(defaultElem)
elem := sl.TryIndex(i).OrZero()

// Map key access
entry := m.TryGet("key").Or(defaultEntry)
entry := m.TryGet("key").OrElse(createDefault)

// Pointer dereference
val := p.TryElem().Or(defaultValue)

// Method lookup
method := s.TryMethod("String").Or(defaultMethod)
```

### Option[T] Methods

| Method | Returns | Description |
|--------|---------|-------------|
| `.Some()` | `bool` | True if value present |
| `.None()` | `bool` | True if empty |
| `.Or(default)` | `T` | Value or default |
| `.OrZero()` | `T` | Value or zero value |
| `.OrElse(fn)` | `T` | Value or lazy default |
| `.Value()` | `(T, bool)` | Traditional Go pattern |
| `.Must()` | `T` | Value or panic |

### Creating Options

```go
opt := gref.Some(value)    // Option containing value
opt := gref.None[Type]()   // Empty Option
```

## Utilities

### Deep Copy

```go
// Copy any value
copied := gref.DeepCopy(original)

// Copy into existing variable
var dest Person
gref.DeepCopyInto(src, &dest)
```

### Deep Equality

```go
gref.Equal(valueA, valueB)     // compare Valuables
reflect.DeepEqual(a, b)        // use stdlib for raw values
```

### Walking / Visiting

```go
// Simple walk - visits all values
gref.Walk(myStruct, func(path string, v gref.Value) bool {
    fmt.Printf("%s: %v\n", path, v.Interface())
    return true // continue walking
})

// Structured visitor - type-specific callbacks
gref.Visit(myStruct, gref.Visitor{
    OnStruct: func(path string, s gref.Struct) bool {
        fmt.Println("Struct at", path)
        return true
    },
    OnSlice: func(path string, s gref.Slice) bool {
        fmt.Println("Slice at", path, "len:", s.Len())
        return true
    },
    OnMap: func(path string, m gref.Map) bool {
        fmt.Println("Map at", path)
        return true
    },
    OnValue: func(path string, v gref.Value) bool {
        fmt.Println("Value at", path, ":", v.Interface())
        return true
    },
})
```

### Struct ↔ Map Conversion

```go
// Struct to map (uses json tags for keys)
m := gref.StructToMap(person, "json")

// Map to struct
var person Person
gref.MapToStruct(data, &person, "json")
```

## Complete Examples

### Example 1: JSON-like Field Access

```go
func getNestedField(v any, path string) (any, error) {
    current := gref.From(v)
    
    for _, part := range strings.Split(path, ".") {
        switch current.Kind() {
        case gref.Struct, gref.Pointer:
            s := current.Struct()
            f, ok := s.TryField(part).Value()
            if !ok {
                return nil, fmt.Errorf("field %q not found", part)
            }
            current = f.Value()
        default:
            return nil, fmt.Errorf("cannot navigate into %s", current.Kind())
        }
    }
    
    return current.Interface(), nil
}

// Usage
user := &User{Address: &Address{City: "Seattle"}}
city, _ := getNestedField(user, "Address.City")
fmt.Println(city) // Seattle
```

### Example 2: Struct Validator

```go
func validate(v any) []string {
    var errors []string
    
    s := gref.From(v).Struct()
    s.Fields().WithTag("validate").Each(func(f gref.Field) bool {
        tag := f.ParsedTag("validate")
        
        if tag.HasOption("required") && f.IsZero() {
            errors = append(errors, f.Name()+" is required")
        }
        
        return true
    })
    
    return errors
}

// Usage
user := &User{Name: "", Age: 30}
errs := validate(user) // ["Name is required"]
```

### Example 3: Generic Map/Filter/Reduce

```go
func filterSlice[T any](slice []T, pred func(T) bool) []T {
    s := gref.From(slice).Slice()
    
    filtered := s.Filter(func(e gref.Element) bool {
        val, _ := gref.Get[T](e)
        return pred(val)
    })
    
    result := make([]T, filtered.Len())
    filtered.Each(func(e gref.Element) bool {
        val, _ := gref.Get[T](e)
        result[e.Position()] = val
        return true
    })
    
    return result
}

// Usage
nums := []int{1, 2, 3, 4, 5}
evens := filterSlice(nums, func(n int) bool { return n%2 == 0 })
// [2, 4]
```

### Example 4: Dynamic Struct Builder

```go
type fieldDef struct {
    Name string
    Type reflect.Type
    Tag  string
}

func buildMap(fields []fieldDef) gref.Map {
    m := gref.MakeMap[string, any]()
    
    for _, f := range fields {
        m = m.Set(f.Name, reflect.Zero(f.Type).Interface())
    }
    
    return m
}
```

### Example 5: Method Dispatcher

```go
func dispatch(obj any, method string, args ...any) ([]any, error) {
    s := gref.From(obj).Struct()
    
    m, ok := s.TryMethod(method).Value()
    if !ok {
        return nil, fmt.Errorf("method %q not found", method)
    }
    
    results := m.Call(args...)
    
    if err := results.Error(); err != nil {
        return nil, err
    }
    
    return results.Collect(), nil
}

// Usage
user := &User{Name: "Alice"}
result, _ := dispatch(user, "Greet")
fmt.Println(result[0]) // "Hello, I'm Alice"
```

## Error Handling Summary

| Operation | Behavior |
|-----------|----------|
| `From(nil)` | Returns invalid Value (check with `.IsValid()`) |
| `.Struct()` on non-struct | **Panics** |
| `.Slice()` on non-slice | **Panics** |
| `.Field("Unknown")` | **Panics** |
| `.TryField("Unknown")` | Returns `Option[Field]` with `.None() == true` |
| `.Index(-1)` | **Panics** |
| `.TryIndex(-1)` | Returns `Option[Element]` with `.None() == true` |
| `Get[int](stringField)` | Returns `(zero, error)` |
| `MustGet[int](stringField)` | **Panics** |
| `TryGet[int](stringField)` | Returns `Result[int]` with `.Ok() == false` |
| `TryGet[int](field).Or(0)` | Returns `0` on failure |

## License

MIT
