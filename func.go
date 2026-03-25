package gref

import (
	"fmt"
	"reflect"
)

// Func provides operations on function values.
// Create via Value.Func().
//
// Example:
//
//	f := refl.From(strings.ToUpper).Func()
//	result := f.Call("hello")
//	s, _ := refl.Get[string](result.First())  // "HELLO"
type Func struct {
	rv reflect.Value
	rt reflect.Type
}

// reflectValue implements Valuable.
func (f Func) reflectValue() reflect.Value {
	return f.rv
}

// --- Type Information ---

// NumIn returns the number of input parameters.
func (f Func) NumIn() int {
	return f.rt.NumIn()
}

// NumOut returns the number of output values.
func (f Func) NumOut() int {
	return f.rt.NumOut()
}

// In returns the type of the i-th input parameter.
func (f Func) In(i int) reflect.Type {
	return f.rt.In(i)
}

// Out returns the type of the i-th output value.
func (f Func) Out(i int) reflect.Type {
	return f.rt.Out(i)
}

// IsVariadic returns true if the function is variadic.
func (f Func) IsVariadic() bool {
	return f.rt.IsVariadic()
}

// Type returns the function type.
func (f Func) Type() reflect.Type {
	return f.rt
}

// IsNil returns true if the function is nil.
func (f Func) IsNil() bool {
	return f.rv.IsNil()
}

// ReturnsError returns true if the last return type is error.
func (f Func) ReturnsError() bool {
	if f.rt.NumOut() == 0 {
		return false
	}
	lastOut := f.rt.Out(f.rt.NumOut() - 1)
	return lastOut.Implements(reflect.TypeOf((*error)(nil)).Elem())
}

// --- Signature ---

// Signature returns a description of the function signature.
//
// Example:
//
//	sig := f.Signature()
//	fmt.Println(sig.String())  // "func(string, int) (bool, error)"
func (f Func) Signature() Signature {
	sig := Signature{
		Variadic: f.rt.IsVariadic(),
		In:       make([]reflect.Type, f.rt.NumIn()),
		Out:      make([]reflect.Type, f.rt.NumOut()),
	}
	for i := 0; i < f.rt.NumIn(); i++ {
		sig.In[i] = f.rt.In(i)
	}
	for i := 0; i < f.rt.NumOut(); i++ {
		sig.Out[i] = f.rt.Out(i)
	}
	return sig
}

// Signature describes a function's input and output types.
type Signature struct {
	Variadic bool
	In       []reflect.Type
	Out      []reflect.Type
}

// String returns a string representation.
func (s Signature) String() string {
	result := "func("
	for i, t := range s.In {
		if i > 0 {
			result += ", "
		}
		if s.Variadic && i == len(s.In)-1 {
			result += "..." + t.Elem().String()
		} else {
			result += t.String()
		}
	}
	result += ")"

	if len(s.Out) > 0 {
		result += " "
		if len(s.Out) > 1 {
			result += "("
		}
		for i, t := range s.Out {
			if i > 0 {
				result += ", "
			}
			result += t.String()
		}
		if len(s.Out) > 1 {
			result += ")"
		}
	}

	return result
}

// --- Invocation ---

// Call invokes the function with the given arguments.
// Returns Results for accessing return values.
//
// Example:
//
//	f := refl.From(strconv.Atoi).Func()
//	r := f.Call("42")
//	n, _ := refl.Get[int](r.First())
//	err := r.Error()
func (f Func) Call(args ...any) Results {
	// Convert args to reflect.Value
	in := make([]reflect.Value, len(args))
	for i, arg := range args {
		if arg == nil {
			// Create typed nil for the expected parameter type
			if i < f.rt.NumIn() {
				in[i] = reflect.Zero(f.rt.In(i))
			} else if f.rt.IsVariadic() && i >= f.rt.NumIn()-1 {
				in[i] = reflect.Zero(f.rt.In(f.rt.NumIn() - 1).Elem())
			} else {
				in[i] = reflect.Zero(reflect.TypeOf((*any)(nil)).Elem())
			}
		} else {
			in[i] = reflect.ValueOf(arg)
		}
	}

	// Call (handle variadic)
	var out []reflect.Value
	if f.rt.IsVariadic() && len(in) >= f.rt.NumIn() {
		out = f.rv.Call(in)
	} else {
		out = f.rv.Call(in)
	}

	return Results{values: out}
}

// CallWith invokes the function with Values as arguments.
func (f Func) CallWith(args ...Value) Results {
	in := make([]reflect.Value, len(args))
	for i, arg := range args {
		in[i] = arg.rv
	}
	out := f.rv.Call(in)
	return Results{values: out}
}

// --- Bind ---

// Bind creates a new function with some arguments pre-filled (partial application).
//
// Example:
//
//	greet := func(greeting, name string) string {
//	    return greeting + ", " + name
//	}
//	hello := refl.From(greet).Func().Bind("Hello")
//	r := hello.Call("World")  // "Hello, World"
func (f Func) Bind(args ...any) Func {
	numBound := len(args)
	numRemaining := f.rt.NumIn() - numBound

	if numRemaining < 0 {
		panic(fmt.Sprintf("gref: too many bound arguments: got %d, function takes %d", numBound, f.rt.NumIn()))
	}

	// Build new input types
	inTypes := make([]reflect.Type, numRemaining)
	for i := 0; i < numRemaining; i++ {
		inTypes[i] = f.rt.In(numBound + i)
	}

	// Build output types
	outTypes := make([]reflect.Type, f.rt.NumOut())
	for i := 0; i < f.rt.NumOut(); i++ {
		outTypes[i] = f.rt.Out(i)
	}

	// Create new function type
	newType := reflect.FuncOf(inTypes, outTypes, false)

	// Pre-convert bound args
	boundArgs := make([]reflect.Value, numBound)
	for i, arg := range args {
		boundArgs[i] = reflect.ValueOf(arg)
	}

	// Create bound function
	impl := func(in []reflect.Value) []reflect.Value {
		fullArgs := append(boundArgs, in...)
		return f.rv.Call(fullArgs)
	}

	rv := reflect.MakeFunc(newType, impl)
	return Func{rv: rv, rt: newType}
}

// --- Conversion ---

// Interface returns the function as any.
func (f Func) Interface() any {
	return f.rv.Interface()
}

// Value returns this function as a Value.
func (f Func) Value() Value {
	return Value{rv: f.rv}
}

// ============================================================================
// Results
// ============================================================================

// Results holds function return values.
// Created via Func.Call().
//
// Example:
//
//	r := f.Call(args...)
//	first, _ := refl.Get[string](r.First())
//	err := r.Error()  // extracts error if last return is error
type Results struct {
	values []reflect.Value
}

// Len returns the number of return values.
func (r Results) Len() int {
	return len(r.values)
}

// Index returns the i-th return value.
// Panics if out of bounds.
func (r Results) Index(i int) Value {
	if i < 0 || i >= len(r.values) {
		panic(fmt.Sprintf("gref: result index %d out of bounds [0:%d]", i, len(r.values)))
	}
	return Value{rv: r.values[i]}
}

// First returns the first return value. Panics if no return values.
func (r Results) First() Value {
	return r.Index(0)
}

// Last returns the last return value. Panics if no return values.
func (r Results) Last() Value {
	return r.Index(len(r.values) - 1)
}

// Error extracts an error from the last return value.
// Returns nil if the last return isn't an error type or is nil error.
//
// Example:
//
//	r := f.Call("invalid")
//	if err := r.Error(); err != nil {
//	    log.Fatal(err)
//	}
func (r Results) Error() error {
	if len(r.values) == 0 {
		return nil
	}

	last := r.values[len(r.values)-1]
	if !last.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		return nil
	}
	if last.IsNil() {
		return nil
	}
	return last.Interface().(error)
}

// All returns all return values.
func (r Results) All() []Value {
	result := make([]Value, len(r.values))
	for i, v := range r.values {
		result[i] = Value{rv: v}
	}
	return result
}

// Collect returns all return values as []any.
func (r Results) Collect() []any {
	result := make([]any, len(r.values))
	for i, v := range r.values {
		result[i] = v.Interface()
	}
	return result
}

// Unpack unpacks return values into pointers.
//
// Example:
//
//	var name string
//	var err error
//	r.Unpack(&name, &err)
func (r Results) Unpack(ptrs ...any) error {
	if len(ptrs) > len(r.values) {
		return fmt.Errorf("gref: want to unpack %d values but only have %d", len(ptrs), len(r.values))
	}

	for i, ptr := range ptrs {
		if ptr == nil {
			continue
		}

		pv := reflect.ValueOf(ptr)
		if pv.Kind() != reflect.Pointer || pv.IsNil() {
			return fmt.Errorf("%w: argument %d must be non-nil pointer", ErrTypeMismatch, i)
		}

		target := pv.Elem()
		val := r.values[i]

		if !val.Type().AssignableTo(target.Type()) {
			if val.Type().ConvertibleTo(target.Type()) {
				target.Set(val.Convert(target.Type()))
				continue
			}
			return fmt.Errorf("%w: cannot assign %s to %s", ErrTypeMismatch, val.Type(), target.Type())
		}

		target.Set(val)
	}

	return nil
}
