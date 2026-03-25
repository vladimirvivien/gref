// Package main demonstrates using gref for struct tag introspection
// to build a custom JSON-like serializer.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/vladimirvivien/gref"
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
// It handles:
//   - Custom field names via `json:"name"`
//   - Omitting fields via `json:"-"`
//   - Omitting zero values via `json:",omitempty"`
func Serialize(v any) map[string]any {
	s := gref.From(v).Struct()
	result := make(map[string]any)

	s.Fields().Each(func(field gref.Field) bool {
		tag := field.ParsedTag("json")

		// Skip fields marked with "-"
		if tag.Name == "-" {
			return true // continue
		}

		// Skip zero values if omitempty is set
		if tag.HasOption("omitempty") && field.IsZero() {
			return true // continue
		}

		// Use tag name or fall back to field name
		key := tag.Name
		if key == "" {
			key = field.Name()
		}

		result[key] = field.Interface()
		return true // continue
	})

	return result
}

// SerializeSimple shows the simpler approach using ToMapTag
func SerializeSimple(v any) map[string]any {
	return gref.From(v).Struct().ToMapTag("json")
}

// SerializeNested handles nested structs recursively
func SerializeNested(v any) map[string]any {
	s := gref.From(v).Struct()
	result := make(map[string]any)

	s.Fields().Each(func(field gref.Field) bool {
		tag := field.ParsedTag("json")

		if tag.Name == "-" {
			return true
		}

		key := tag.Name
		if key == "" {
			key = field.Name()
		}

		if tag.HasOption("omitempty") && field.IsZero() {
			return true
		}

		// Recursively serialize nested structs
		if field.Kind() == gref.StructKind {
			result[key] = SerializeNested(field.Interface())
		} else {
			result[key] = field.Interface()
		}
		return true
	})

	return result
}

func main() {
	user := User{
		ID:        1,
		Username:  "alice",
		Email:     "alice@example.com",
		Password:  "secret123", // Should be omitted due to json:"-"
		FirstName: "",          // Should be omitted due to omitempty
		LastName:  "Smith",
		Active:    true,
	}

	fmt.Println("=== gref JSON Serializer Example ===")
	fmt.Println()

	// Serialize using gref
	serialized := Serialize(&user)

	fmt.Println("Original struct:")
	fmt.Printf("  %+v\n", user)
	fmt.Println()

	fmt.Println("Serialized with gref (respecting json tags):")
	output, _ := json.MarshalIndent(serialized, "  ", "  ")
	fmt.Printf("  %s\n", output)
	fmt.Println()

	// Show what standard json.Marshal produces (for comparison)
	fmt.Println("Standard json.Marshal (for comparison):")
	standard, _ := json.MarshalIndent(user, "  ", "  ")
	fmt.Printf("  %s\n", standard)
	fmt.Println()

	// Demonstrate nested struct handling
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
			// Zip is empty, should be omitted
		},
	}

	fmt.Println("Nested struct serialization:")
	nested := SerializeNested(&person)
	nestedOutput, _ := json.MarshalIndent(nested, "  ", "  ")
	fmt.Printf("  %s\n", nestedOutput)

	os.Exit(0)
}
