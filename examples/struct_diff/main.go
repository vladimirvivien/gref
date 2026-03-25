// Package main demonstrates using gref to compare structs and track changes.
// Useful for audit logs, change detection, and dirty checking in ORMs.
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/vladimirvivien/gref"
)

// Change represents a single field change
type Change struct {
	Path     string
	OldValue any
	NewValue any
	Type     gref.Type
}

func (c Change) String() string {
	return fmt.Sprintf("%s: %v -> %v", c.Path, c.OldValue, c.NewValue)
}

// Diff compares two structs and returns the differences
func Diff(old, new any) []Change {
	var changes []Change
	diffRecursive(gref.From(old).Struct(), gref.From(new).Struct(), "", &changes)
	return changes
}

func diffRecursive(oldS, newS gref.Struct, prefix string, changes *[]Change) {
	oldS.Fields().Exported().Each(func(oldField gref.Field) bool {
		fieldName := oldField.Name()
		path := fieldName
		if prefix != "" {
			path = prefix + "." + fieldName
		}

		newField, ok := newS.TryField(fieldName).Value()
		if !ok {
			return true
		}

		// Handle nested structs (but not time.Time)
		if oldField.Kind() == gref.StructKind && !isTimeType(oldField.Type()) {
			diffRecursive(oldField.Struct(), newField.Struct(), path, changes)
			return true
		}

		// Compare values using gref.Equal
		oldVal := gref.From(oldField.Interface())
		newVal := gref.From(newField.Interface())

		if !gref.Equal(oldVal, newVal) {
			*changes = append(*changes, Change{
				Path:     path,
				OldValue: oldField.Interface(),
				NewValue: newField.Interface(),
				Type:     oldField.Type(),
			})
		}

		return true
	})
}

func isTimeType(t gref.Type) bool {
	return t == gref.TypeOf[time.Time]()
}

// ChangeTracker wraps a struct and tracks modifications
type ChangeTracker struct {
	original any
	current  any
}

// Track creates a change tracker for a struct
func Track(v any) *ChangeTracker {
	// Deep copy the original
	original := gref.DeepCopy(v)
	return &ChangeTracker{
		original: original,
		current:  v,
	}
}

// Changes returns all changes since tracking started
func (ct *ChangeTracker) Changes() []Change {
	return Diff(ct.original, ct.current)
}

// HasChanges returns true if there are any changes
func (ct *ChangeTracker) HasChanges() bool {
	return len(ct.Changes()) > 0
}

// Reset clears the change tracking (current becomes new baseline)
func (ct *ChangeTracker) Reset() {
	ct.original = gref.DeepCopy(ct.current)
}

// Revert restores the struct to its original state
func (ct *ChangeTracker) Revert() {
	// Copy original values back to current
	gref.DeepCopyInto(ct.original, ct.current)
}

// ApplyChanges applies a set of changes to a struct
func ApplyChanges(target any, changes []Change) {
	s := gref.From(target).Struct()
	for _, change := range changes {
		if field, ok := s.TryField(change.Path).Value(); ok {
			field.Set(change.NewValue)
		}
	}
}

// --- Example structs ---

type Address struct {
	Street  string
	City    string
	Country string
	ZipCode string
}

type User struct {
	ID        int
	Name      string
	Email     string
	Age       int
	Active    bool
	Address   Address
	Tags      []string
	UpdatedAt time.Time
}

func main() {
	fmt.Println("=== gref Struct Diff Example ===")
	fmt.Println()

	// Create two versions of a user
	userV1 := User{
		ID:     1,
		Name:   "Alice",
		Email:  "alice@example.com",
		Age:    30,
		Active: true,
		Address: Address{
			Street:  "123 Main St",
			City:    "Seattle",
			Country: "USA",
			ZipCode: "98101",
		},
		Tags:      []string{"developer", "gopher"},
		UpdatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	userV2 := User{
		ID:     1,
		Name:   "Alice Smith", // Changed
		Email:  "alice.smith@example.com", // Changed
		Age:    31, // Changed
		Active: true,
		Address: Address{
			Street:  "456 Oak Ave", // Changed
			City:    "Portland", // Changed
			Country: "USA",
			ZipCode: "97201", // Changed
		},
		Tags:      []string{"developer", "gopher", "speaker"}, // Changed
		UpdatedAt: time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC), // Changed
	}

	// Compare the two versions
	fmt.Println("1. Comparing two struct versions:")
	changes := Diff(userV1, userV2)
	for _, c := range changes {
		fmt.Printf("   %s\n", c)
	}
	fmt.Println()

	// Use change tracker for live tracking
	fmt.Println("2. Live change tracking:")
	user := &User{
		ID:    2,
		Name:  "Bob",
		Email: "bob@example.com",
		Age:   25,
		Address: Address{
			City: "New York",
		},
	}

	tracker := Track(user)
	fmt.Printf("   Initial state: Name=%s, Age=%d\n", user.Name, user.Age)
	fmt.Printf("   Has changes: %v\n", tracker.HasChanges())

	// Make some changes
	user.Name = "Robert"
	user.Age = 26
	user.Email = "robert@example.com"
	user.Address.City = "Boston"

	fmt.Printf("\n   After modifications:\n")
	fmt.Printf("   Has changes: %v\n", tracker.HasChanges())
	fmt.Println("   Changes detected:")
	for _, c := range tracker.Changes() {
		fmt.Printf("     %s\n", c)
	}

	// Demonstrate revert
	fmt.Println("\n3. Reverting changes:")
	fmt.Printf("   Before revert: Name=%s, Age=%d\n", user.Name, user.Age)
	tracker.Revert()
	fmt.Printf("   After revert: Name=%s, Age=%d\n", user.Name, user.Age)
	fmt.Printf("   Has changes: %v\n", tracker.HasChanges())

	// Generate audit log
	fmt.Println("\n4. Generating audit log:")
	userV1.Name = "Alice Smith"
	userV1.Email = "alice.smith@example.com"

	changes = Diff(User{ID: 1, Name: "Alice", Email: "alice@example.com"}, userV1)
	fmt.Println("   Audit entry:")
	fmt.Println("   {")
	fmt.Println("     \"entity\": \"User\",")
	fmt.Println("     \"id\": 1,")
	fmt.Println("     \"changes\": [")
	for i, c := range changes {
		comma := ","
		if i == len(changes)-1 {
			comma = ""
		}
		fmt.Printf("       {\"field\": \"%s\", \"old\": %q, \"new\": %q}%s\n",
			c.Path, fmt.Sprint(c.OldValue), fmt.Sprint(c.NewValue), comma)
	}
	fmt.Println("     ]")
	fmt.Println("   }")

	// Filter changes by path prefix
	fmt.Println("\n5. Filtering changes by path:")
	changes = Diff(
		User{Address: Address{Street: "A", City: "B", ZipCode: "C"}},
		User{Address: Address{Street: "X", City: "Y", ZipCode: "Z"}},
	)
	fmt.Println("   Address-only changes:")
	for _, c := range changes {
		if strings.HasPrefix(c.Path, "Address.") {
			fmt.Printf("     %s\n", c)
		}
	}
}
