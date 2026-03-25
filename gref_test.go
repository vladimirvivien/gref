package gref_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/vladimirvivien/gref"
)

// ============================================================================
// Test Types
// ============================================================================

type Address struct {
	Street  string `json:"street" validate:"required"`
	City    string `json:"city"`
	ZipCode string `json:"zip_code,omitempty"`
}

type Person struct {
	Name    string            `json:"name" validate:"required"`
	Age     int               `json:"age"`
	Email   string            `json:"email,omitempty"`
	Address *Address          `json:"address"`
	Tags    []string          `json:"tags"`
	Scores  map[string]int    `json:"scores"`
}

type Greeter interface {
	Greet() string
}

func (p Person) Greet() string {
	return fmt.Sprintf("Hello, I'm %s", p.Name)
}

func (p *Person) SetName(name string) {
	p.Name = name
}

// ============================================================================
// Value Tests
// ============================================================================

func TestFrom(t *testing.T) {
	// Basic value
	v := gref.From(42)
	if v.Kind() != gref.Int {
		t.Errorf("expected Int, got %v", v.Kind())
	}

	// Pointer (should NOT auto-deref at Value level)
	x := 42
	v = gref.From(&x)
	if v.Kind() != gref.Pointer {
		t.Errorf("expected Pointer, got %v", v.Kind())
	}

	// Nil
	v = gref.From(nil)
	if v.IsValid() {
		t.Error("expected invalid for nil")
	}
}

func TestValueNavigation(t *testing.T) {
	user := &Person{
		Name: "Alice",
		Age:  30,
		Address: &Address{
			City: "Seattle",
		},
	}

	// Auto-deref when navigating to Struct
	s := gref.From(user).Struct()
	if s.NumFields() != 6 {
		t.Errorf("expected 6 fields, got %d", s.NumFields())
	}
}

func TestValuePanicOnWrongKind(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for wrong type")
		}
	}()

	gref.From(42).Struct() // Should panic - int is not a struct
}

// ============================================================================
// Struct and Field Tests
// ============================================================================

func TestStructField(t *testing.T) {
	user := &Person{
		Name: "Alice",
		Age:  30,
	}

	s := gref.From(user).Struct()

	// Get field
	nameField := s.Field("Name")
	name, err := gref.Get[string](nameField)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "Alice" {
		t.Errorf("expected 'Alice', got %s", name)
	}

	// Field metadata
	if nameField.Tag("json") != "name" {
		t.Errorf("expected json tag 'name', got %s", nameField.Tag("json"))
	}
	if nameField.Tag("validate") != "required" {
		t.Errorf("expected validate tag 'required', got %s", nameField.Tag("validate"))
	}
}

func TestStructFieldPanicOnNotFound(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for unknown field")
		}
	}()

	user := &Person{Name: "Alice"}
	gref.From(user).Struct().Field("Unknown") // Should panic
}

func TestStructDotNotation(t *testing.T) {
	user := &Person{
		Name: "Alice",
		Address: &Address{
			City:   "Seattle",
			Street: "123 Main St",
		},
	}

	s := gref.From(user).Struct()

	// Dot notation
	cityField := s.Field("Address.City")
	city, err := gref.Get[string](cityField)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if city != "Seattle" {
		t.Errorf("expected 'Seattle', got %s", city)
	}

	// Explicit chaining (same result)
	city2, _ := gref.Get[string](s.Field("Address").Struct().Field("City"))
	if city != city2 {
		t.Errorf("dot notation and explicit chaining should match")
	}
}

func TestStructFieldIteration(t *testing.T) {
	user := &Person{Name: "Alice", Age: 30}
	s := gref.From(user).Struct()

	var names []string
	s.Fields().Each(func(f gref.Field) bool {
		names = append(names, f.Name())
		return true
	})

	if len(names) != 6 {
		t.Errorf("expected 6 fields, got %d: %v", len(names), names)
	}

	// Filter by tag
	var withValidate []string
	s.Fields().WithTag("validate").Each(func(f gref.Field) bool {
		withValidate = append(withValidate, f.Name())
		return true
	})
	if len(withValidate) != 1 {
		t.Errorf("expected 1 field with validate tag, got %d", len(withValidate))
	}
}

func TestStructParsedTag(t *testing.T) {
	user := &Person{}
	s := gref.From(user).Struct()

	tag := s.Field("Email").ParsedTag("json")
	if !tag.Exists {
		t.Fatal("expected tag to exist")
	}
	if tag.Name != "email" {
		t.Errorf("expected name 'email', got %s", tag.Name)
	}
	if !tag.HasOption("omitempty") {
		t.Error("expected omitempty option")
	}
}

func TestStructToMap(t *testing.T) {
	user := Person{
		Name: "Alice",
		Age:  30,
	}

	m := gref.From(&user).Struct().ToMapTag("json")

	if m["name"] != "Alice" {
		t.Errorf("expected 'Alice', got %v", m["name"])
	}
	if m["age"] != 30 {
		t.Errorf("expected 30, got %v", m["age"])
	}
}

func TestStructSetField(t *testing.T) {
	user := &Person{Name: "Alice"}
	s := gref.From(user).Struct()

	s.SetField("Name", "Bob")
	if user.Name != "Bob" {
		t.Errorf("expected 'Bob', got %s", user.Name)
	}
}

func TestStructMethod(t *testing.T) {
	user := &Person{Name: "Alice"}
	s := gref.From(user).Struct()

	// Value receiver method
	greet := s.Method("Greet")
	results := greet.Call()
	greeting, _ := gref.Get[string](results.First())
	if greeting != "Hello, I'm Alice" {
		t.Errorf("expected greeting, got %s", greeting)
	}
}

func TestStructTryField(t *testing.T) {
	user := &Person{Name: "Alice"}
	s := gref.From(user).Struct()

	// Existing field - using Value()
	f, ok := s.TryField("Name").Value()
	if !ok {
		t.Error("expected Name field to exist")
	}
	if f.Name() != "Name" {
		t.Errorf("expected 'Name', got %s", f.Name())
	}

	// Non-existing field - using None()
	if s.TryField("Unknown").Some() {
		t.Error("expected Unknown field to not exist")
	}
	
	// Using Or() for default
	defaultField := s.Field("Age")
	field := s.TryField("Unknown").Or(defaultField)
	if field.Name() != "Age" {
		t.Errorf("expected default field 'Age', got %s", field.Name())
	}
}

// ============================================================================
// Slice and Element Tests
// ============================================================================

func TestSliceBasics(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5}
	s := gref.From(nums).Slice()

	if s.Len() != 5 {
		t.Errorf("expected len 5, got %d", s.Len())
	}

	// Index access
	elem := s.Index(2)
	val, _ := gref.Get[int](elem)
	if val != 3 {
		t.Errorf("expected 3, got %d", val)
	}

	// Element position
	if elem.Position() != 2 {
		t.Errorf("expected position 2, got %d", elem.Position())
	}
}

func TestSliceIndexPanicOnOutOfBounds(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for out of bounds")
		}
	}()

	nums := []int{1, 2, 3}
	gref.From(nums).Slice().Index(10) // Should panic
}

func TestSliceAppend(t *testing.T) {
	nums := []int{1, 2, 3}
	s := gref.From(nums).Slice()

	s = s.Append(4, 5)
	if s.Len() != 5 {
		t.Errorf("expected len 5, got %d", s.Len())
	}
}

func TestSliceFilter(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5}
	s := gref.From(nums).Slice()

	evens := s.Filter(func(e gref.Element) bool {
		val, _ := gref.Get[int](e)
		return val%2 == 0
	})

	if evens.Len() != 2 {
		t.Errorf("expected 2 evens, got %d", evens.Len())
	}
}

func TestSliceMap(t *testing.T) {
	nums := []int{1, 2, 3}
	s := gref.From(nums).Slice()

	doubled := s.Map(func(e gref.Element) any {
		val, _ := gref.Get[int](e)
		return val * 2
	})

	first, _ := gref.Get[int](doubled.First())
	if first != 2 {
		t.Errorf("expected 2, got %d", first)
	}
}

func TestSliceReduce(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5}
	s := gref.From(nums).Slice()

	sum := s.Reduce(0, func(acc any, e gref.Element) any {
		val, _ := gref.Get[int](e)
		return acc.(int) + val
	})

	if sum.(int) != 15 {
		t.Errorf("expected 15, got %v", sum)
	}
}

func TestSliceOfStructs(t *testing.T) {
	users := []Person{
		{Name: "Alice", Age: 30},
		{Name: "Bob", Age: 25},
	}

	s := gref.From(users).Slice()

	// Navigate into struct element
	firstUser := s.First().Struct()
	name, _ := gref.Get[string](firstUser.Field("Name"))
	if name != "Alice" {
		t.Errorf("expected 'Alice', got %s", name)
	}
}

// ============================================================================
// Map and Entry Tests
// ============================================================================

func TestMapBasics(t *testing.T) {
	data := map[string]int{"a": 1, "b": 2, "c": 3}
	m := gref.From(data).Map()

	if m.Len() != 3 {
		t.Errorf("expected len 3, got %d", m.Len())
	}

	// Get entry
	entry := m.Get("b")
	val, _ := gref.Get[int](entry)
	if val != 2 {
		t.Errorf("expected 2, got %d", val)
	}

	// Entry key
	key, _ := gref.Get[string](entry.Key())
	if key != "b" {
		t.Errorf("expected 'b', got %s", key)
	}

	// Has
	if !m.Has("a") {
		t.Error("expected 'a' to exist")
	}
}

func TestMapGetPanicOnNotFound(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for key not found")
		}
	}()

	data := map[string]int{"a": 1}
	gref.From(data).Map().Get("z") // Should panic
}

func TestMapSetDelete(t *testing.T) {
	data := map[string]int{"a": 1}
	m := gref.From(data).Map()

	m = m.Set("b", 2)
	if !m.Has("b") {
		t.Error("expected 'b' after set")
	}

	m = m.Delete("a")
	if m.Has("a") {
		t.Error("expected 'a' to be deleted")
	}
}

func TestMapFilter(t *testing.T) {
	data := map[string]int{"a": 1, "b": 2, "c": 3}
	m := gref.From(data).Map()

	filtered := m.Filter(func(e gref.Entry) bool {
		val, _ := gref.Get[int](e)
		return val > 1
	})

	if filtered.Len() != 2 {
		t.Errorf("expected 2, got %d", filtered.Len())
	}
}

func TestMapOfStructs(t *testing.T) {
	users := map[string]Person{
		"alice": {Name: "Alice", Age: 30},
		"bob":   {Name: "Bob", Age: 25},
	}

	m := gref.From(users).Map()

	// Navigate into struct value
	alice := m.Get("alice").Struct()
	age, _ := gref.Get[int](alice.Field("Age"))
	if age != 30 {
		t.Errorf("expected 30, got %d", age)
	}
}

func TestMapToStructMethod(t *testing.T) {
	type User struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	// With json tags
	data1 := map[string]any{"id": 1, "name": "Alice", "age": 30}
	m1 := gref.From(data1).Map()

	var u1 User
	m1.ToStructTag(&u1, "json")

	if u1.ID != 1 || u1.Name != "Alice" || u1.Age != 30 {
		t.Errorf("expected {1 Alice 30}, got %+v", u1)
	}

	// With field names (no tag)
	data2 := map[string]any{"ID": 2, "Name": "Bob", "Age": 25}
	m2 := gref.From(data2).Map()

	var u2 User
	m2.ToStruct(&u2)

	if u2.ID != 2 || u2.Name != "Bob" || u2.Age != 25 {
		t.Errorf("expected {2 Bob 25}, got %+v", u2)
	}

	// With type conversion (int to int64, etc.)
	type Stats struct {
		Count int64
		Rate  float64
	}

	data3 := map[string]any{"Count": 100, "Rate": 3.14}
	var s Stats
	gref.From(data3).Map().ToStruct(&s)

	if s.Count != 100 || s.Rate != 3.14 {
		t.Errorf("expected {100 3.14}, got %+v", s)
	}
}

// ============================================================================
// Channel Tests
// ============================================================================

func TestChanBasics(t *testing.T) {
	ch := make(chan int, 3)
	c := gref.From(ch).Chan()

	if c.Cap() != 3 {
		t.Errorf("expected cap 3, got %d", c.Cap())
	}

	c.Send(1).Send(2).Send(3)

	if c.Len() != 3 {
		t.Errorf("expected len 3, got %d", c.Len())
	}

	val, ok := c.Recv()
	if !ok {
		t.Error("expected successful receive")
	}
	n, _ := gref.Get[int](val)
	if n != 1 {
		t.Errorf("expected 1, got %d", n)
	}
}

// ============================================================================
// Function Tests
// ============================================================================

func TestFuncCall(t *testing.T) {
	add := func(a, b int) int { return a + b }

	f := gref.From(add).Func()
	results := f.Call(2, 3)

	sum, _ := gref.Get[int](results.First())
	if sum != 5 {
		t.Errorf("expected 5, got %d", sum)
	}
}

func TestFuncWithError(t *testing.T) {
	mayFail := func(ok bool) (string, error) {
		if ok {
			return "success", nil
		}
		return "", errors.New("failed")
	}

	f := gref.From(mayFail).Func()

	// Success
	r := f.Call(true)
	if r.Error() != nil {
		t.Errorf("expected no error: %v", r.Error())
	}

	// Failure
	r = f.Call(false)
	if r.Error() == nil {
		t.Error("expected error")
	}
}

func TestFuncSignature(t *testing.T) {
	fn := func(s string, n int) (bool, error) { return true, nil }
	f := gref.From(fn).Func()

	if f.NumIn() != 2 {
		t.Errorf("expected 2 inputs, got %d", f.NumIn())
	}
	if f.NumOut() != 2 {
		t.Errorf("expected 2 outputs, got %d", f.NumOut())
	}
	if !f.ReturnsError() {
		t.Error("expected function to return error")
	}

	sig := f.Signature()
	expected := "func(string, int) (bool, error)"
	if sig.String() != expected {
		t.Errorf("expected %q, got %q", expected, sig.String())
	}
}

func TestFuncBind(t *testing.T) {
	greet := func(greeting, name string) string {
		return greeting + ", " + name
	}

	f := gref.From(greet).Func()
	hello := f.Bind("Hello")

	r := hello.Call("World")
	result, _ := gref.Get[string](r.First())
	if result != "Hello, World" {
		t.Errorf("expected 'Hello, World', got %s", result)
	}
}

// ============================================================================
// Pointer Tests
// ============================================================================

func TestPtrBasics(t *testing.T) {
	x := 42
	p := gref.From(&x).Ptr()

	if p.IsNil() {
		t.Error("expected non-nil")
	}

	val, _ := gref.Get[int](p.Elem())
	if val != 42 {
		t.Errorf("expected 42, got %d", val)
	}
}

func TestPtrSet(t *testing.T) {
	x := 42
	p := gref.From(&x).Ptr()

	p.Set(100)
	if x != 100 {
		t.Errorf("expected 100, got %d", x)
	}
}

func TestPtrDerefAll(t *testing.T) {
	x := 42
	ptr := &x
	ptrptr := &ptr

	p := gref.From(ptrptr).Ptr()

	if p.IndirectionDepth() != 2 {
		t.Errorf("expected depth 2, got %d", p.IndirectionDepth())
	}

	val, _ := gref.Get[int](p.DerefAll())
	if val != 42 {
		t.Errorf("expected 42, got %d", val)
	}
}

// ============================================================================
// Get[T] Tests
// ============================================================================

func TestGet(t *testing.T) {
	user := &Person{Name: "Alice", Age: 30}
	s := gref.From(user).Struct()

	// Correct type
	name, err := gref.Get[string](s.Field("Name"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "Alice" {
		t.Errorf("expected 'Alice', got %s", name)
	}

	// Type mismatch - should error, not panic
	_, err = gref.Get[int](s.Field("Name"))
	if err == nil {
		t.Error("expected error for type mismatch")
	}

	// Convertible type (int → int64)
	age64, err := gref.Get[int64](s.Field("Age"))
	if err != nil {
		t.Fatalf("unexpected error for convertible type: %v", err)
	}
	if age64 != 30 {
		t.Errorf("expected 30, got %d", age64)
	}
}

func TestMustGet(t *testing.T) {
	user := &Person{Name: "Alice"}
	name := gref.MustGet[string](gref.From(user).Struct().Field("Name"))
	if name != "Alice" {
		t.Errorf("expected 'Alice', got %s", name)
	}
}

func TestMustGetPanicsOnTypeMismatch(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for type mismatch")
		}
	}()

	user := &Person{Name: "Alice"}
	gref.MustGet[int](gref.From(user).Struct().Field("Name")) // Should panic
}

func TestTryGet(t *testing.T) {
	user := &Person{Name: "Alice"}
	s := gref.From(user).Struct()

	// Using .Value() for (val, ok) pattern
	if name, ok := gref.TryGet[string](s.Field("Name")).Value(); ok {
		if name != "Alice" {
			t.Errorf("expected 'Alice', got %s", name)
		}
	} else {
		t.Error("expected TryGet to succeed")
	}

	// Using .Ok() for just checking
	if gref.TryGet[int](s.Field("Name")).Ok() {
		t.Error("expected TryGet to fail for wrong type")
	}
}

// ============================================================================
// Make[T] Tests (Unified API)
// ============================================================================

func TestMakeSlice(t *testing.T) {
	// New unified API - args on Slice()
	s := gref.Make[[]int]().Slice(3, 10)

	if s.Len() != 3 {
		t.Errorf("expected len 3, got %d", s.Len())
	}
	if s.Cap() != 10 {
		t.Errorf("expected cap 10, got %d", s.Cap())
	}

	s = s.Set(0, 42)
	val, _ := gref.Get[int](s.First())
	if val != 42 {
		t.Errorf("expected 42, got %d", val)
	}
}

func TestMakeMap(t *testing.T) {
	// New unified API
	m := gref.Make[map[string]int]().Map()

	m = m.Set("a", 1).Set("b", 2)

	if m.Len() != 2 {
		t.Errorf("expected len 2, got %d", m.Len())
	}

	val, _ := gref.Get[int](m.Get("a"))
	if val != 1 {
		t.Errorf("expected 1, got %d", val)
	}
}

func TestMakeChan(t *testing.T) {
	// New unified API - args on Chan()
	c := gref.Make[chan string]().Chan(5)

	if c.Cap() != 5 {
		t.Errorf("expected cap 5, got %d", c.Cap())
	}

	c.Send("hello")
	val, _ := c.Recv()
	s, _ := gref.Get[string](val)
	if s != "hello" {
		t.Errorf("expected 'hello', got %s", s)
	}
}

func TestMakeStruct(t *testing.T) {
	// New unified API
	s := gref.Make[Person]().Struct()
	
	s.SetField("Name", "Alice")
	s.SetField("Age", 30)
	
	name, _ := gref.Get[string](s.Field("Name"))
	if name != "Alice" {
		t.Errorf("expected 'Alice', got %s", name)
	}
	
	age, _ := gref.Get[int](s.Field("Age"))
	if age != 30 {
		t.Errorf("expected 30, got %d", age)
	}
}

func TestMakeFunc(t *testing.T) {
	// New unified API
	f := gref.Make[func(string, int) bool]().Impl(
		func(name string, age int) bool {
			return age >= 18
		},
	)
	
	result := f.Call("Alice", 25)
	isAdult, _ := gref.Get[bool](result.First())
	if !isAdult {
		t.Error("expected true for age 25")
	}
	
	result = f.Call("Bob", 15)
	isAdult, _ = gref.Get[bool](result.First())
	if isAdult {
		t.Error("expected false for age 15")
	}
}

func TestMakeDefaults(t *testing.T) {
	// Test default arguments
	s := gref.Make[[]int]().Slice() // len=0, cap=0
	if s.Len() != 0 {
		t.Errorf("expected len 0, got %d", s.Len())
	}
	
	c := gref.Make[chan int]().Chan() // unbuffered
	if c.Cap() != 0 {
		t.Errorf("expected cap 0, got %d", c.Cap())
	}
}

func TestMakerType(t *testing.T) {
	// Test getting Type from Maker
	maker := gref.Make[[]int]()
	rt := maker.Type()
	
	if rt.Kind() != reflect.Slice {
		t.Errorf("expected slice kind, got %s", rt.Kind())
	}
}

// ============================================================================
// Runtime Def Tests
// ============================================================================

func TestDefSlice(t *testing.T) {
	// New unified Def() API
	s := gref.Def().Slice(gref.IntType, 5, 10)

	if s.Len() != 5 {
		t.Errorf("expected len 5, got %d", s.Len())
	}
	if s.Cap() != 10 {
		t.Errorf("expected cap 10, got %d", s.Cap())
	}
	
	s = s.Append(42)
	if s.Len() != 6 {
		t.Errorf("expected len 6 after append, got %d", s.Len())
	}
}

func TestDefMap(t *testing.T) {
	// New unified Def() API
	m := gref.Def().Map(gref.StringType, gref.IntType)

	m = m.Set("x", 99)
	if m.Len() != 1 {
		t.Errorf("expected len 1, got %d", m.Len())
	}
	
	val, _ := gref.Get[int](m.Get("x"))
	if val != 99 {
		t.Errorf("expected 99, got %d", val)
	}
}

func TestDefChan(t *testing.T) {
	// New unified Def() API
	c := gref.Def().Chan(gref.StringType, 3)

	if c.Cap() != 3 {
		t.Errorf("expected cap 3, got %d", c.Cap())
	}
	
	c.Send("hello")
	val, _ := c.Recv()
	s, _ := gref.Get[string](val)
	if s != "hello" {
		t.Errorf("expected 'hello', got %s", s)
	}
}

func TestDefStruct(t *testing.T) {
	// New unified Def() API - no Build() needed
	s := gref.Def().Struct("DynamicPerson").
		Field(gref.FieldDef{Name: "Name", Type: gref.StringType, Tag: `json:"name"`}).
		Field(gref.FieldDef{Name: "Age", Type: gref.IntType, Tag: `json:"age"`}).
		Set("Name", "Alice").
		Set("Age", 30).
		Struct()

	name, _ := gref.Get[string](s.Field("Name"))
	if name != "Alice" {
		t.Errorf("expected 'Alice', got %s", name)
	}
	
	age, _ := gref.Get[int](s.Field("Age"))
	if age != 30 {
		t.Errorf("expected 30, got %d", age)
	}
	
	// Check tag
	tag := s.Field("Name").Tag("json")
	if tag != "name" {
		t.Errorf("expected tag 'name', got %s", tag)
	}
}

func TestDefFunc(t *testing.T) {
	// New unified Def() API - Impl() returns Func directly, no Build() needed
	fn := gref.Def().Func("isAdult").
		Arg(gref.ArgDef{Name: "name", Type: gref.StringType}).
		Arg(gref.ArgDef{Name: "age", Type: gref.IntType}).
		Return(gref.ReturnDef{Type: gref.BoolType}).
		Impl(func(args []gref.Value) []gref.Value {
			age, _ := gref.Get[int](args[1])
			return []gref.Value{gref.From(age >= 18)}
		})

	result := fn.Call("Alice", 25)
	isAdult, _ := gref.Get[bool](result.First())
	if !isAdult {
		t.Error("expected true for age 25")
	}

	result = fn.Call("Bob", 15)
	isAdult, _ = gref.Get[bool](result.First())
	if isAdult {
		t.Error("expected false for age 15")
	}
}

func TestStructFrom(t *testing.T) {
	// Create from existing struct
	user := gref.Make[Person]().Struct()
	user.SetField("Name", "Alice")
	user.SetField("Age", 30)
	
	// Create modified copy
	copy := gref.StructFrom(user).Set("Name", "Bob").Struct()
	
	copyName, _ := gref.Get[string](copy.Field("Name"))
	if copyName != "Bob" {
		t.Errorf("expected 'Bob', got %s", copyName)
	}
	
	// Original unchanged
	origName, _ := gref.Get[string](user.Field("Name"))
	if origName != "Alice" {
		t.Errorf("original should be 'Alice', got %s", origName)
	}
}

// Legacy API tests (for backward compatibility)

func TestSliceDefLegacy(t *testing.T) {
	s := gref.SliceDef(gref.IntType, 5, 10)

	if s.Len() != 5 {
		t.Errorf("expected len 5, got %d", s.Len())
	}
	if s.Cap() != 10 {
		t.Errorf("expected cap 10, got %d", s.Cap())
	}
}

func TestMapDefLegacy(t *testing.T) {
	m := gref.MapDef(gref.StringType, gref.IntType)

	m = m.Set("x", 99)
	if m.Len() != 1 {
		t.Errorf("expected len 1, got %d", m.Len())
	}
}

func TestChanDefLegacy(t *testing.T) {
	c := gref.ChanDef(gref.StringType, 3)

	if c.Cap() != 3 {
		t.Errorf("expected cap 3, got %d", c.Cap())
	}
}

func TestNew(t *testing.T) {
	p := gref.New[Person]()

	if p.IsNil() {
		t.Error("expected non-nil pointer")
	}

	s := p.Struct()
	name, _ := gref.Get[string](s.Field("Name"))
	if name != "" {
		t.Errorf("expected empty string (zero value), got %s", name)
	}
}

func TestMakeStructLegacy(t *testing.T) {
	// Test legacy MakeStruct[T]() API
	user := gref.MakeStruct[Person]()
	
	user.SetField("Name", "Alice")
	user.SetField("Age", 30)
	
	name, _ := gref.Get[string](user.Field("Name"))
	if name != "Alice" {
		t.Errorf("expected 'Alice', got %s", name)
	}
	
	age, _ := gref.Get[int](user.Field("Age"))
	if age != 30 {
		t.Errorf("expected 30, got %d", age)
	}
}

func TestMakeFuncLegacy(t *testing.T) {
	// Test legacy MakeFunc[F]() API
	f := gref.MakeFunc[func(string, int) bool]().Impl(
		func(name string, age int) bool {
			return age >= 18
		},
	)
	
	result := f.Call("Alice", 25)
	isAdult, _ := gref.Get[bool](result.First())
	if !isAdult {
		t.Error("expected true for age 25")
	}
	
	result = f.Call("Bob", 15)
	isAdult, _ = gref.Get[bool](result.First())
	if isAdult {
		t.Error("expected false for age 15")
	}
}

// ============================================================================
// TryGet Result Tests
// ============================================================================

func TestTryGetOr(t *testing.T) {
	s := gref.From(Person{Name: "Alice", Age: 30}).Struct()
	
	// Success case
	name := gref.TryGet[string](s.Field("Name")).Or("default")
	if name != "Alice" {
		t.Errorf("expected 'Alice', got %s", name)
	}
	
	// Failure case (wrong type)
	wrongType := gref.TryGet[int](s.Field("Name")).Or(42)
	if wrongType != 42 {
		t.Errorf("expected default 42, got %d", wrongType)
	}
}

func TestTryGetOrZero(t *testing.T) {
	s := gref.From(Person{Name: "Alice", Age: 30}).Struct()
	
	// Success case
	age := gref.TryGet[int](s.Field("Age")).OrZero()
	if age != 30 {
		t.Errorf("expected 30, got %d", age)
	}
	
	// Failure case
	wrongType := gref.TryGet[string](s.Field("Age")).OrZero()
	if wrongType != "" {
		t.Errorf("expected empty string, got %s", wrongType)
	}
}

func TestTryGetOrElse(t *testing.T) {
	s := gref.From(Person{Name: "Alice"}).Struct()
	
	called := false
	result := gref.TryGet[int](s.Field("Name")).OrElse(func() int {
		called = true
		return 99
	})
	
	if !called {
		t.Error("expected OrElse function to be called")
	}
	if result != 99 {
		t.Errorf("expected 99, got %d", result)
	}
	
	// Success case - should NOT call the function
	called = false
	result = gref.TryGet[int](s.Field("Age")).OrElse(func() int {
		called = true
		return 99
	})
	
	if called {
		t.Error("expected OrElse function NOT to be called on success")
	}
}

func TestTryGetOk(t *testing.T) {
	s := gref.From(Person{Name: "Alice"}).Struct()
	
	if !gref.TryGet[string](s.Field("Name")).Ok() {
		t.Error("expected Ok() to be true for string field")
	}
	
	if gref.TryGet[int](s.Field("Name")).Ok() {
		t.Error("expected Ok() to be false for wrong type")
	}
}

func TestTryGetValue(t *testing.T) {
	s := gref.From(Person{Name: "Alice"}).Struct()
	
	val, ok := gref.TryGet[string](s.Field("Name")).Value()
	if !ok {
		t.Error("expected ok to be true")
	}
	if val != "Alice" {
		t.Errorf("expected 'Alice', got %s", val)
	}
	
	_, ok = gref.TryGet[int](s.Field("Name")).Value()
	if ok {
		t.Error("expected ok to be false for wrong type")
	}
}

// ============================================================================
// Is[T] Tests
// ============================================================================

func TestIs(t *testing.T) {
	v := gref.From(42)
	
	if !gref.Is[int](v) {
		t.Error("expected Is[int] to be true")
	}
	
	// int can be converted to int64
	if !gref.Is[int64](v) {
		t.Error("expected Is[int64] to be true (convertible)")
	}
	
	if gref.Is[string](v) {
		t.Error("expected Is[string] to be false")
	}
}

func TestIsExactly(t *testing.T) {
	v := gref.From(int64(42))
	
	if !gref.IsExactly[int64](v) {
		t.Error("expected IsExactly[int64] to be true")
	}
	
	// int64 is NOT exactly int (even though convertible)
	if gref.IsExactly[int](v) {
		t.Error("expected IsExactly[int] to be false")
	}
}

// ============================================================================
// Option[T] Tests for Try* Methods
// ============================================================================

func TestOptionTryIndex(t *testing.T) {
	sl := gref.From([]int{10, 20, 30}).Slice()
	
	// Valid index - using Value()
	elem, ok := sl.TryIndex(1).Value()
	if !ok {
		t.Error("expected index 1 to exist")
	}
	val, _ := gref.Get[int](elem)
	if val != 20 {
		t.Errorf("expected 20, got %d", val)
	}
	
	// Invalid index - using None()
	if sl.TryIndex(10).Some() {
		t.Error("expected index 10 to not exist")
	}
	
	// Using Or() for default
	defaultElem := sl.Index(0)
	elem = sl.TryIndex(10).Or(defaultElem)
	val, _ = gref.Get[int](elem)
	if val != 10 {
		t.Errorf("expected default 10, got %d", val)
	}
	
	// Using OrZero()
	zeroElem := sl.TryIndex(10).OrZero()
	// Zero Element should have zero index
	if zeroElem.Position() != 0 {
		t.Errorf("expected zero Element with position 0, got %d", zeroElem.Position())
	}
}

func TestOptionMapTryGet(t *testing.T) {
	m := gref.From(map[string]int{"a": 1, "b": 2}).Map()
	
	// Existing key - using Value()
	entry, ok := m.TryGet("a").Value()
	if !ok {
		t.Error("expected key 'a' to exist")
	}
	val, _ := gref.Get[int](entry)
	if val != 1 {
		t.Errorf("expected 1, got %d", val)
	}
	
	// Non-existing key - using None()
	if m.TryGet("unknown").Some() {
		t.Error("expected key 'unknown' to not exist")
	}
	
	// Using Or()
	defaultEntry := m.Get("b")
	entry = m.TryGet("unknown").Or(defaultEntry)
	val, _ = gref.Get[int](entry)
	if val != 2 {
		t.Errorf("expected default 2, got %d", val)
	}
}

func TestOptionTryElem(t *testing.T) {
	x := 42
	p := gref.From(&x).Ptr()
	
	// Non-nil pointer - using Value()
	val, ok := p.TryElem().Value()
	if !ok {
		t.Error("expected non-nil pointer to have element")
	}
	n, _ := gref.Get[int](val)
	if n != 42 {
		t.Errorf("expected 42, got %d", n)
	}
	
	// Nil pointer
	var nilPtr *int
	p = gref.From(nilPtr).Ptr()
	if p.TryElem().Some() {
		t.Error("expected nil pointer to have no element")
	}
}

func TestOptionTryMethod(t *testing.T) {
	s := gref.From(Person{Name: "Alice"}).Struct()
	
	// Non-existing method
	if s.TryMethod("NonExistent").Some() {
		t.Error("expected method 'NonExistent' to not exist")
	}
}

func TestOptionSomeNone(t *testing.T) {
	// Test Some()
	opt := gref.Some(42)
	if !opt.Some() {
		t.Error("expected Some() to be true")
	}
	if opt.None() {
		t.Error("expected None() to be false")
	}
	
	// Test None()
	opt2 := gref.None[int]()
	if opt2.Some() {
		t.Error("expected Some() to be false for None")
	}
	if !opt2.None() {
		t.Error("expected None() to be true for None")
	}
}

func TestOptionMust(t *testing.T) {
	opt := gref.Some(42)
	if opt.Must() != 42 {
		t.Error("expected Must() to return 42")
	}
	
	// Test panic on None
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected Must() to panic on None")
		}
	}()
	gref.None[int]().Must()
}

// ============================================================================
// Iterator Tests (Go 1.23+ range-over-func)
// ============================================================================

func TestFieldIterator(t *testing.T) {
	s := gref.From(Person{Name: "Alice", Age: 30}).Struct()
	
	var names []string
	for field := range s.Fields().Iter() {
		names = append(names, field.Name())
	}
	
	if len(names) != 2 {
		t.Errorf("expected 2 fields, got %d", len(names))
	}
	if names[0] != "Name" || names[1] != "Age" {
		t.Errorf("unexpected field names: %v", names)
	}
}

func TestFilteredFieldIterator(t *testing.T) {
	type Tagged struct {
		Name string `json:"name"`
		Age  int    
		City string `json:"city"`
	}
	
	s := gref.From(Tagged{}).Struct()
	
	var tagged []string
	for field := range s.Fields().WithTag("json").Iter() {
		tagged = append(tagged, field.Name())
	}
	
	if len(tagged) != 2 {
		t.Errorf("expected 2 tagged fields, got %d", len(tagged))
	}
}

func TestSliceIterator(t *testing.T) {
	sl := gref.From([]int{10, 20, 30}).Slice()
	
	var sum int
	for elem := range sl.Iter() {
		val, _ := gref.Get[int](elem)
		sum += val
	}
	
	if sum != 60 {
		t.Errorf("expected sum 60, got %d", sum)
	}
}

func TestMapIterator(t *testing.T) {
	m := gref.From(map[string]int{"a": 1, "b": 2, "c": 3}).Map()
	
	var sum int
	for entry := range m.Iter() {
		val, _ := gref.Get[int](entry)
		sum += val
	}
	
	if sum != 6 {
		t.Errorf("expected sum 6, got %d", sum)
	}
}

func TestIteratorEarlyBreak(t *testing.T) {
	sl := gref.From([]int{1, 2, 3, 4, 5}).Slice()
	
	var count int
	for range sl.Iter() {
		count++
		if count == 3 {
			break
		}
	}
	
	if count != 3 {
		t.Errorf("expected count 3 after break, got %d", count)
	}
}

func TestStructDefLegacy(t *testing.T) {
	// Legacy API - Create a struct at runtime with dynamic fields
	st := gref.StructDef("DynamicPerson").
		FieldDef(gref.FieldDef{Name: "Name", Type: gref.StringType, Tag: `json:"name"`}).
		FieldDef(gref.FieldDef{Name: "Age", Type: gref.IntType, Tag: `json:"age"`}).
		Set("Name", "Alice").
		Set("Age", 30)
	
	s := st.Build()
	
	name, _ := gref.Get[string](s.Field("Name"))
	if name != "Alice" {
		t.Errorf("expected 'Alice', got %s", name)
	}
	
	age, _ := gref.Get[int](s.Field("Age"))
	if age != 30 {
		t.Errorf("expected 30, got %d", age)
	}
	
	// Check tag
	tag := s.Field("Name").Tag("json")
	if tag != "name" {
		t.Errorf("expected json tag 'name', got %s", tag)
	}
}

func TestStructDefFromLegacy(t *testing.T) {
	// Legacy API - Create a struct definition from existing compile-time struct
	user := gref.MakeStruct[Person]()
	user.SetField("Name", "Alice")
	user.SetField("Age", 30)
	
	st := gref.StructDefFrom(user)
	st.Set("Name", "Bob")
	
	s := st.Build()
	name, _ := gref.Get[string](s.Field("Name"))
	if name != "Bob" {
		t.Errorf("expected 'Bob', got %s", name)
	}
	
	// Original should be unchanged (StructDefFrom makes a copy)
	origName, _ := gref.Get[string](user.Field("Name"))
	if origName != "Alice" {
		t.Errorf("expected original 'Alice', got %s", origName)
	}
}

func TestFuncDefLegacy(t *testing.T) {
	// Legacy API - Create a function at runtime with dynamic signature
	fn := gref.FuncDef("isAdult").
		ArgDef(gref.ArgDef{Name: "name", Type: gref.StringType}).
		ArgDef(gref.ArgDef{Name: "age", Type: gref.IntType}).
		ReturnDef(gref.ReturnDef{Type: gref.BoolType}).
		Impl(func(args []gref.Value) []gref.Value {
			age, _ := gref.Get[int](args[1])
			return []gref.Value{gref.From(age >= 18)}
		}).
		Build()
	
	result := fn.Call("Alice", 25)
	isAdult, _ := gref.Get[bool](result.First())
	if !isAdult {
		t.Error("expected true for age 25")
	}
}

func TestSlicePtr(t *testing.T) {
	// Test that MakeSlice can chain to Ptr
	s := gref.MakeSlice[int](0, 10).Append(1, 2, 3)
	ptr := s.Ptr()
	
	slicePtr, ok := ptr.(*[]int)
	if !ok {
		t.Fatalf("expected *[]int, got %T", ptr)
	}
	
	if len(*slicePtr) != 3 {
		t.Errorf("expected len 3, got %d", len(*slicePtr))
	}
	if (*slicePtr)[0] != 1 || (*slicePtr)[1] != 2 || (*slicePtr)[2] != 3 {
		t.Errorf("expected [1,2,3], got %v", *slicePtr)
	}
}

func TestPredefinedTypes(t *testing.T) {
	// Test that predefined types work correctly
	if gref.StringType.Kind() != reflect.String {
		t.Error("StringType should be string")
	}
	if gref.IntType.Kind() != reflect.Int {
		t.Error("IntType should be int")
	}
	if gref.BoolType.Kind() != reflect.Bool {
		t.Error("BoolType should be bool")
	}
	if gref.Float64Type.Kind() != reflect.Float64 {
		t.Error("Float64Type should be float64")
	}
}

// ============================================================================
// Utility Tests
// ============================================================================

func TestDeepCopy(t *testing.T) {
	original := Person{
		Name: "Alice",
		Age:  30,
		Address: &Address{
			City: "Seattle",
		},
		Tags: []string{"dev", "go"},
	}

	copied := gref.DeepCopy(original).(Person)

	// Modify original
	original.Name = "Bob"
	original.Address.City = "Portland"
	original.Tags[0] = "ops"

	// Copy should be unchanged
	if copied.Name != "Alice" {
		t.Errorf("expected 'Alice', got %s", copied.Name)
	}
	if copied.Address.City != "Seattle" {
		t.Errorf("expected 'Seattle', got %s", copied.Address.City)
	}
	if copied.Tags[0] != "dev" {
		t.Errorf("expected 'dev', got %s", copied.Tags[0])
	}
}

func TestWalk(t *testing.T) {
	user := Person{
		Name: "Alice",
		Address: &Address{
			City: "Seattle",
		},
	}

	var paths []string
	gref.Walk(&user, func(path string, v gref.Value) bool {
		paths = append(paths, path)
		return true
	})

	// Should include nested paths
	found := false
	for _, path := range paths {
		if path == "Address.City" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'Address.City' in paths: %v", paths)
	}
}

func TestStructToMapHelper(t *testing.T) {
	user := Person{Name: "Alice", Age: 30}
	m := gref.StructToMap(&user, "json")

	if m["name"] != "Alice" {
		t.Errorf("expected 'Alice', got %v", m["name"])
	}
}

func TestMapToStruct(t *testing.T) {
	m := map[string]any{
		"name": "Alice",
		"age":  30,
	}

	var user Person
	err := gref.MapToStruct(m, &user, "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.Name != "Alice" {
		t.Errorf("expected 'Alice', got %s", user.Name)
	}
}

// ============================================================================
// Complex Chaining Examples
// ============================================================================

func TestComplexChaining(t *testing.T) {
	type Company struct {
		Name      string
		Employees []Person
		Offices   map[string]Address
	}

	company := Company{
		Name: "Acme",
		Employees: []Person{
			{Name: "Alice", Age: 30, Address: &Address{City: "Seattle"}},
			{Name: "Bob", Age: 25, Address: &Address{City: "Portland"}},
		},
		Offices: map[string]Address{
			"HQ": {City: "Seattle"},
		},
	}

	// Navigate: Company → Employees[0] → Address → City
	city, err := gref.Get[string](
		gref.From(&company).Struct().
			Field("Employees").Slice().
			First().Struct().
			Field("Address.City"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if city != "Seattle" {
		t.Errorf("expected 'Seattle', got %s", city)
	}

	// Navigate: Company → Offices["HQ"] → City
	hqCity, err := gref.Get[string](
		gref.From(&company).Struct().
			Field("Offices").Map().
			Get("HQ").Struct().
			Field("City"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hqCity != "Seattle" {
		t.Errorf("expected 'Seattle', got %s", hqCity)
	}
}

// ============================================================================
// Documentation Examples
// ============================================================================

func ExampleFrom() {
	user := struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}{
		Name: "Alice",
		Age:  30,
	}

	// Navigate to struct and get field
	s := gref.From(&user).Struct()
	name, _ := gref.Get[string](s.Field("Name"))
	fmt.Println(name)

	// Access field tag
	tag := s.Field("Name").Tag("json")
	fmt.Println(tag)

	// Output:
	// Alice
	// name
}

func ExampleMakeMap() {
	// Create a map with compile-time known types
	m := gref.MakeMap[string, int]()
	m = m.Set("one", 1).Set("two", 2)

	val, _ := gref.Get[int](m.Get("one"))
	fmt.Println(val)

	// Output:
	// 1
}

func ExampleGet() {
	data := map[string]any{
		"count": 42,
	}

	m := gref.From(data).Map()
	count, err := gref.Get[int](m.Get("count"))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(count)

	// Output:
	// 42
}

// ============================================================================
// Type Construction Tests
// ============================================================================

func TestTypeOf(t *testing.T) {
	intType := gref.TypeOf[int]()
	if intType.Kind() != reflect.Int {
		t.Errorf("expected Int, got %v", intType.Kind())
	}

	stringType := gref.TypeOf[string]()
	if stringType.Kind() != reflect.String {
		t.Errorf("expected String, got %v", stringType.Kind())
	}
}

func TestTypeConstruction(t *testing.T) {
	// Build a map type at runtime
	mapType := gref.MapTypeOf(
		gref.TypeOf[string](),
		gref.TypeOf[int](),
	)

	if mapType.Kind() != reflect.Map {
		t.Errorf("expected Map, got %v", mapType.Kind())
	}
	if mapType.Key().Kind() != reflect.String {
		t.Errorf("expected string key, got %v", mapType.Key().Kind())
	}
}
