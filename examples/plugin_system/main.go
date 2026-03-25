// Package main demonstrates using gref for dynamic function creation,
// useful for building plugin systems, middleware, and decorators.
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/vladimirvivien/gref"
)

// WrapWithLogging wraps any function with logging that records
// arguments, return values, and execution time.
func WrapWithLogging(name string, fn any) gref.Func {
	original := gref.From(fn).Func()

	// Build a function with the same signature
	builder := gref.Def().Func(name + "_logged")

	// Copy input types
	for i := 0; i < original.NumIn(); i++ {
		builder.Arg(gref.ArgDef{Type: original.In(i)})
	}

	// Copy output types
	for i := 0; i < original.NumOut(); i++ {
		builder.Return(gref.ReturnDef{Type: original.Out(i)})
	}

	// Handle variadic functions
	if original.IsVariadic() {
		builder.Variadic()
	}

	// Create the wrapper implementation
	return builder.Impl(func(args []gref.Value) []gref.Value {
		// Log the call
		argStrs := make([]string, len(args))
		for i, arg := range args {
			argStrs[i] = fmt.Sprintf("%v", arg.Interface())
		}
		fmt.Printf("[LOG] Calling %s(%s)\n", name, strings.Join(argStrs, ", "))

		// Execute and time the original function
		start := time.Now()
		results := original.CallWith(args...)
		elapsed := time.Since(start)

		// Log the results
		allResults := results.All()
		resultStrs := make([]string, len(allResults))
		for i, r := range allResults {
			resultStrs[i] = fmt.Sprintf("%v", r.Interface())
		}
		fmt.Printf("[LOG] %s returned (%s) in %v\n", name, strings.Join(resultStrs, ", "), elapsed)

		return allResults
	})
}

// WrapWithRetry wraps a function to retry on error (last return must be error)
func WrapWithRetry(name string, fn any, maxRetries int) gref.Func {
	original := gref.From(fn).Func()

	// Verify last return is error
	numOut := original.NumOut()
	if numOut == 0 || original.Out(numOut-1) != gref.ErrorType {
		panic("WrapWithRetry requires function with error as last return value")
	}

	builder := gref.Def().Func(name + "_retry")

	for i := 0; i < original.NumIn(); i++ {
		builder.Arg(gref.ArgDef{Type: original.In(i)})
	}
	for i := 0; i < original.NumOut(); i++ {
		builder.Return(gref.ReturnDef{Type: original.Out(i)})
	}

	return builder.Impl(func(args []gref.Value) []gref.Value {
		var results gref.Results

		for attempt := 1; attempt <= maxRetries; attempt++ {
			results = original.CallWith(args...)

			// Check if last result (error) is nil
			lastResult := results.Last()
			if lastResult.IsNil() {
				return results.All()
			}

			err := lastResult.Interface().(error)
			fmt.Printf("[RETRY] %s attempt %d/%d failed: %v\n", name, attempt, maxRetries, err)

			if attempt < maxRetries {
				time.Sleep(time.Millisecond * 100 * time.Duration(attempt)) // Exponential backoff
			}
		}

		return results.All()
	})
}

// CreateMiddlewareChain creates a chain of middleware functions
func CreateMiddlewareChain(handler any, middlewares ...func(gref.Func) gref.Func) gref.Func {
	fn := gref.From(handler).Func()

	// Apply middlewares in reverse order (so first middleware is outermost)
	for i := len(middlewares) - 1; i >= 0; i-- {
		fn = middlewares[i](fn)
	}

	return fn
}

// TimingMiddleware adds timing to any function
func TimingMiddleware(next gref.Func) gref.Func {
	builder := gref.Def().Func("timed")

	for i := 0; i < next.NumIn(); i++ {
		builder.Arg(gref.ArgDef{Type: next.In(i)})
	}
	for i := 0; i < next.NumOut(); i++ {
		builder.Return(gref.ReturnDef{Type: next.Out(i)})
	}

	return builder.Impl(func(args []gref.Value) []gref.Value {
		start := time.Now()
		results := next.CallWith(args...)
		fmt.Printf("[TIMING] Execution took %v\n", time.Since(start))
		return results.All()
	})
}

// Example functions to wrap

func Add(a, b int) int {
	return a + b
}

func Greet(name string) string {
	return fmt.Sprintf("Hello, %s!", name)
}

func Divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, fmt.Errorf("division by zero")
	}
	return a / b, nil
}

// Simulated unreliable function
var callCount = 0

func UnreliableOperation(data string) (string, error) {
	callCount++
	if callCount < 3 {
		return "", fmt.Errorf("temporary failure (attempt %d)", callCount)
	}
	return fmt.Sprintf("processed: %s", data), nil
}

func main() {
	fmt.Println("=== gref Plugin System Example ===")
	fmt.Println()

	// Example 1: Wrap functions with logging
	fmt.Println("1. Function wrapping with logging:")
	fmt.Println()

	loggedAdd := WrapWithLogging("Add", Add)
	result := loggedAdd.Call(5, 3)
	fmt.Printf("   Result: %v\n", result.First().Interface())
	fmt.Println()

	loggedGreet := WrapWithLogging("Greet", Greet)
	greetResult := loggedGreet.Call("World")
	fmt.Printf("   Result: %v\n", greetResult.First().Interface())
	fmt.Println()

	// Example 2: Retry wrapper
	fmt.Println("2. Retry wrapper for unreliable operations:")
	fmt.Println()

	callCount = 0 // Reset
	retryOp := WrapWithRetry("UnreliableOperation", UnreliableOperation, 5)
	opResult := retryOp.Call("test-data")
	fmt.Printf("   Final result: %v\n", opResult.First().Interface())
	fmt.Println()

	// Example 3: Function introspection
	fmt.Println("3. Function introspection:")
	fmt.Println()

	inspectFunc := func(name string, fn any) {
		f := gref.From(fn).Func()
		fmt.Printf("   %s:\n", name)
		fmt.Printf("     Inputs:  ")
		for i := 0; i < f.NumIn(); i++ {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(f.In(i))
		}
		fmt.Println()
		fmt.Printf("     Outputs: ")
		for i := 0; i < f.NumOut(); i++ {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(f.Out(i))
		}
		fmt.Println()
		fmt.Printf("     Variadic: %v\n", f.IsVariadic())
	}

	inspectFunc("Add", Add)
	inspectFunc("Greet", Greet)
	inspectFunc("Divide", Divide)
	inspectFunc("fmt.Printf", fmt.Printf)
	fmt.Println()

	// Example 4: Create function at runtime
	fmt.Println("4. Runtime function creation:")
	fmt.Println()

	// Create a string transformer function at runtime
	transformer := gref.Def().Func("transform").
		Arg(gref.ArgDef{Type: gref.StringType}).
		Return(gref.ReturnDef{Type: gref.StringType}).
		Impl(func(args []gref.Value) []gref.Value {
			input := args[0].String()
			output := strings.ToUpper(input) + "!"
			return []gref.Value{gref.From(output)}
		})

	transformResult := transformer.Call("hello")
	fmt.Printf("   transform(\"hello\") = %v\n", transformResult.First().Interface())
	fmt.Println()

	// Example 5: Dynamic dispatch based on type
	fmt.Println("5. Dynamic dispatch (type-based routing):")
	fmt.Println()

	handlers := map[gref.Type]func(any) string{
		gref.IntType:    func(v any) string { return fmt.Sprintf("integer: %d", v) },
		gref.StringType: func(v any) string { return fmt.Sprintf("string: %q", v) },
		gref.BoolType:   func(v any) string { return fmt.Sprintf("boolean: %v", v) },
	}

	dispatch := func(v any) string {
		val := gref.From(v)
		if handler, ok := handlers[val.Type()]; ok {
			return handler(v)
		}
		return fmt.Sprintf("unknown type: %T", v)
	}

	fmt.Printf("   dispatch(42) = %s\n", dispatch(42))
	fmt.Printf("   dispatch(\"test\") = %s\n", dispatch("test"))
	fmt.Printf("   dispatch(true) = %s\n", dispatch(true))
	fmt.Printf("   dispatch(3.14) = %s\n", dispatch(3.14))
	fmt.Println()

	// Example 6: Middleware chain
	fmt.Println("6. Middleware chain:")
	fmt.Println()

	// Reset for clean output
	processor := func(data string) string {
		time.Sleep(10 * time.Millisecond) // Simulate work
		return "processed: " + data
	}

	chainedProcessor := CreateMiddlewareChain(processor, TimingMiddleware)
	chainResult := chainedProcessor.Call("input-data")
	fmt.Printf("   Result: %v\n", chainResult.First().Interface())
}
