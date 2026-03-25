// Package main demonstrates the same JSON serializer using stdlib reflect.
// Compare with json_serializer which uses gref.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
)

// User represents a user with JSON tags
type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email,omitempty"`
	Password  string `json:"-"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Active    bool   `json:"active"`
}

// Serialize converts a struct to map[string]any respecting json tags.
func Serialize(v any) map[string]any {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	rt := rv.Type()

	result := make(map[string]any)

	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		fv := rv.Field(i)

		// Parse json tag manually
		name, opts := parseTag(sf.Tag.Get("json"))

		// Skip fields marked with "-"
		if name == "-" {
			continue
		}

		// Use tag name or fall back to field name
		key := name
		if key == "" {
			key = sf.Name
		}

		// Skip zero values if omitempty is set
		if hasOption(opts, "omitempty") && fv.IsZero() {
			continue
		}

		result[key] = fv.Interface()
	}

	return result
}

// SerializeNested handles nested structs recursively
func SerializeNested(v any) map[string]any {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	rt := rv.Type()

	result := make(map[string]any)

	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		fv := rv.Field(i)

		name, opts := parseTag(sf.Tag.Get("json"))

		if name == "-" {
			continue
		}

		key := name
		if key == "" {
			key = sf.Name
		}

		if hasOption(opts, "omitempty") && fv.IsZero() {
			continue
		}

		// Recursively serialize nested structs
		if fv.Kind() == reflect.Struct {
			result[key] = SerializeNested(fv.Interface())
		} else {
			result[key] = fv.Interface()
		}
	}

	return result
}

// parseTag splits a tag into name and options
func parseTag(tag string) (string, []string) {
	parts := strings.Split(tag, ",")
	if len(parts) == 0 {
		return "", nil
	}
	return parts[0], parts[1:]
}

// hasOption checks if an option exists in the options slice
func hasOption(opts []string, opt string) bool {
	for _, o := range opts {
		if o == opt {
			return true
		}
	}
	return false
}

func main() {
	user := User{
		ID:        1,
		Username:  "alice",
		Email:     "alice@example.com",
		Password:  "secret123",
		FirstName: "",
		LastName:  "Smith",
		Active:    true,
	}

	fmt.Println("=== stdlib reflect JSON Serializer Example ===")
	fmt.Println()

	serialized := Serialize(&user)

	fmt.Println("Original struct:")
	fmt.Printf("  %+v\n", user)
	fmt.Println()

	fmt.Println("Serialized with reflect (respecting json tags):")
	output, _ := json.MarshalIndent(serialized, "  ", "  ")
	fmt.Printf("  %s\n", output)
	fmt.Println()

	fmt.Println("Standard json.Marshal (for comparison):")
	standard, _ := json.MarshalIndent(user, "  ", "  ")
	fmt.Printf("  %s\n", standard)
	fmt.Println()

	type Address struct {
		Street string `json:"street"`
		City   string `json:"city"`
		Zip    string `json:"zip,omitempty"`
	}

	type Person struct {
		Name    string  `json:"name"`
		Address Address `json:"address"`
	}

	person := Person{
		Name: "Bob",
		Address: Address{
			Street: "123 Main St",
			City:   "Springfield",
		},
	}

	fmt.Println("Nested struct serialization:")
	nested := SerializeNested(&person)
	nestedOutput, _ := json.MarshalIndent(nested, "  ", "  ")
	fmt.Printf("  %s\n", nestedOutput)

	os.Exit(0)
}
