// Package main demonstrates using gref for API response transformation.
// It shows field filtering, renaming, and computed fields for API responses.
package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/vladimirvivien/gref"
)

// TransformOptions configures how fields are transformed
type TransformOptions struct {
	Include    []string          // Fields to include (empty = all)
	Exclude    []string          // Fields to exclude
	Rename     map[string]string // Field renames: original -> new name
	Computed   map[string]func(any) any // Computed fields
	OmitEmpty  bool              // Omit zero-value fields
	OmitFields []string          // Always omit these fields
}

// Transform converts a struct to a map with transformations applied
func Transform(v any, opts TransformOptions) map[string]any {
	s := gref.From(v).Struct()
	result := make(map[string]any)

	// Build include/exclude sets
	includeSet := make(map[string]bool)
	for _, f := range opts.Include {
		includeSet[f] = true
	}
	excludeSet := make(map[string]bool)
	for _, f := range opts.Exclude {
		excludeSet[f] = true
	}
	for _, f := range opts.OmitFields {
		excludeSet[f] = true
	}

	s.Fields().Exported().Each(func(field gref.Field) bool {
		name := field.Name()

		// Check include list (if specified)
		if len(includeSet) > 0 && !includeSet[name] {
			return true
		}

		// Check exclude list
		if excludeSet[name] {
			return true
		}

		// Skip zero values if OmitEmpty
		if opts.OmitEmpty && field.IsZero() {
			return true
		}

		// Apply rename
		outputName := name
		if newName, ok := opts.Rename[name]; ok {
			outputName = newName
		}

		// Handle nested structs
		if field.Kind() == gref.StructKind && !isSpecialType(field.Type()) {
			nested := Transform(field.Interface(), TransformOptions{OmitEmpty: opts.OmitEmpty})
			if len(nested) > 0 {
				result[outputName] = nested
			}
		} else {
			result[outputName] = field.Interface()
		}

		return true
	})

	// Add computed fields
	for name, fn := range opts.Computed {
		result[name] = fn(v)
	}

	return result
}

func isSpecialType(t gref.Type) bool {
	// Don't recurse into time.Time
	return t.String() == "time.Time"
}

// FieldSelector parses a field selection string like "id,name,email"
func FieldSelector(fields string) []string {
	if fields == "" {
		return nil
	}
	parts := strings.Split(fields, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// APIResponse wraps data with metadata
type APIResponse struct {
	Data    any    `json:"data"`
	Version string `json:"api_version"`
}

// --- Example models ---

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"` // Sensitive - should be excluded
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Age          int       `json:"age"`
	IsAdmin      bool      `json:"is_admin"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Profile      Profile   `json:"profile"`
}

type Profile struct {
	Bio       string   `json:"bio"`
	Website   string   `json:"website"`
	AvatarURL string   `json:"avatar_url"`
	Skills    []string `json:"skills"`
}

func main() {
	fmt.Println("=== gref API Transformer Example ===")
	fmt.Println()

	user := User{
		ID:           1,
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: "bcrypt$2a$10$...", // Should never be exposed
		FirstName:    "Alice",
		LastName:     "Smith",
		Age:          30,
		IsAdmin:      false,
		CreatedAt:    time.Date(2023, 1, 15, 10, 30, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2024, 6, 20, 15, 45, 0, 0, time.UTC),
		Profile: Profile{
			Bio:       "Software developer",
			Website:   "https://alice.dev",
			AvatarURL: "https://avatars.example.com/alice.jpg",
			Skills:    []string{"Go", "Rust", "Python"},
		},
	}

	// 1. Basic transformation - exclude sensitive fields
	fmt.Println("1. Exclude sensitive fields:")
	result := Transform(user, TransformOptions{
		OmitFields: []string{"PasswordHash"},
	})
	printJSON(result)

	// 2. Select specific fields only
	fmt.Println("2. Select specific fields (sparse fieldset):")
	result = Transform(user, TransformOptions{
		Include: FieldSelector("ID, Username, Email"),
	})
	printJSON(result)

	// 3. Rename fields for API compatibility
	fmt.Println("3. Rename fields for external API:")
	result = Transform(user, TransformOptions{
		Include: FieldSelector("ID, Username, Email, FirstName, LastName"),
		Rename: map[string]string{
			"ID":        "user_id",
			"FirstName": "given_name",
			"LastName":  "family_name",
		},
	})
	printJSON(result)

	// 4. Add computed fields
	fmt.Println("4. Add computed fields:")
	result = Transform(user, TransformOptions{
		Include: FieldSelector("ID, Username, FirstName, LastName"),
		Computed: map[string]func(any) any{
			"full_name": func(v any) any {
				u := v.(User)
				return u.FirstName + " " + u.LastName
			},
			"profile_url": func(v any) any {
				u := v.(User)
				return fmt.Sprintf("https://example.com/users/%s", u.Username)
			},
		},
	})
	printJSON(result)

	// 5. Full API response with versioning
	fmt.Println("5. Full API response transformation:")
	transformed := Transform(user, TransformOptions{
		Exclude: []string{"PasswordHash", "IsAdmin"},
		Rename: map[string]string{
			"CreatedAt": "created",
			"UpdatedAt": "modified",
		},
		Computed: map[string]func(any) any{
			"display_name": func(v any) any {
				u := v.(User)
				return fmt.Sprintf("%s %s (@%s)", u.FirstName, u.LastName, u.Username)
			},
		},
	})

	response := APIResponse{
		Data:    transformed,
		Version: "v2",
	}
	printJSON(response)

	// 6. Demonstrate omit empty
	fmt.Println("6. Omit empty fields:")
	sparseUser := User{
		ID:       2,
		Username: "bob",
		// Most fields are zero-value
	}
	result = Transform(sparseUser, TransformOptions{
		OmitEmpty:  true,
		OmitFields: []string{"PasswordHash"},
	})
	printJSON(result)

	// 7. Different views for same data
	fmt.Println("7. Different views for different consumers:")

	// Public view
	fmt.Println("   Public view:")
	public := Transform(user, TransformOptions{
		Include: FieldSelector("Username, Profile"),
	})
	printJSON(public)

	// Admin view
	fmt.Println("   Admin view:")
	admin := Transform(user, TransformOptions{
		Exclude: []string{"PasswordHash"},
	})
	printJSON(admin)
}

func printJSON(v any) {
	data, _ := json.MarshalIndent(v, "   ", "  ")
	fmt.Printf("   %s\n\n", data)
}
