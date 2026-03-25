// Package main demonstrates using gref to map between different struct types.
// Useful for DTOs, API models, and separating internal/external representations.
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/vladimirvivien/gref"
)

// Mapper maps fields between struct types
type Mapper struct {
	mappings   map[string]string        // source field -> dest field
	transforms map[string]func(any) any // field transformers
	ignoreCase bool
}

// NewMapper creates a new struct mapper
func NewMapper() *Mapper {
	return &Mapper{
		mappings:   make(map[string]string),
		transforms: make(map[string]func(any) any),
	}
}

// Map adds an explicit field mapping
func (m *Mapper) Map(source, dest string) *Mapper {
	m.mappings[source] = dest
	return m
}

// Transform adds a field transformer
func (m *Mapper) Transform(field string, fn func(any) any) *Mapper {
	m.transforms[field] = fn
	return m
}

// IgnoreCase enables case-insensitive field matching
func (m *Mapper) IgnoreCase() *Mapper {
	m.ignoreCase = true
	return m
}

// Copy copies fields from source to dest based on mappings and name matching
func (m *Mapper) Copy(source, dest any) error {
	srcStruct := gref.From(source).Struct()
	dstStruct := gref.From(dest).Struct()

	// Build dest field lookup
	dstFields := make(map[string]gref.Field)
	dstStruct.Fields().Exported().Each(func(f gref.Field) bool {
		name := f.Name()
		if m.ignoreCase {
			name = strings.ToLower(name)
		}
		dstFields[name] = f
		return true
	})

	// Copy fields
	srcStruct.Fields().Exported().Each(func(srcField gref.Field) bool {
		srcName := srcField.Name()

		// Determine destination field name
		dstName := srcName
		if mapped, ok := m.mappings[srcName]; ok {
			dstName = mapped
		}

		lookupName := dstName
		if m.ignoreCase {
			lookupName = strings.ToLower(dstName)
		}

		dstField, ok := dstFields[lookupName]
		if !ok {
			return true // Skip if no matching dest field
		}

		if !dstField.CanSet() {
			return true
		}

		// Get source value
		value := srcField.Interface()

		// Apply transform if exists
		if transform, ok := m.transforms[srcName]; ok {
			value = transform(value)
		}

		// Only copy if types match exactly
		srcType := gref.From(value).Type()
		dstType := dstField.Type()

		if srcType == dstType {
			dstField.Set(value)
		} else if srcField.Kind() == gref.StructKind && dstField.Kind() == gref.StructKind {
			// Recursively map nested structs
			m.Copy(value, dstField.Addr().Interface())
		}

		return true
	})

	return nil
}

// AutoMap automatically maps between structs using matching field names
func AutoMap(source, dest any) error {
	return NewMapper().Copy(source, dest)
}

// AutoMapTag maps using struct tags for field matching
func AutoMapTag(source, dest any, tagName string) error {
	srcStruct := gref.From(source).Struct()
	dstStruct := gref.From(dest).Struct()

	// Build dest field lookup by tag value
	dstByTag := make(map[string]gref.Field)
	dstStruct.Fields().Exported().Each(func(f gref.Field) bool {
		tag := f.ParsedTag(tagName)
		if tag.Name != "" && tag.Name != "-" {
			dstByTag[tag.Name] = f
		}
		return true
	})

	// Copy by matching tags
	srcStruct.Fields().Exported().Each(func(srcField gref.Field) bool {
		tag := srcField.ParsedTag(tagName)
		if tag.Name == "" || tag.Name == "-" {
			return true
		}

		dstField, ok := dstByTag[tag.Name]
		if !ok || !dstField.CanSet() {
			return true
		}

		// Only copy if types match
		srcType := srcField.Type()
		dstType := dstField.Type()

		if srcType == dstType {
			dstField.Set(srcField.Interface())
		}

		return true
	})

	return nil
}

// --- Example: Entity to DTO mapping ---

// Internal domain model
type UserEntity struct {
	ID           int64
	Email        string
	PasswordHash string
	FirstName    string
	LastName     string
	DateOfBirth  time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	IsAdmin      bool
	LoginCount   int
}

// Public API response
type UserDTO struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	FullName  string `json:"full_name"`
	Age       int    `json:"age"`
	MemberFor string `json:"member_for"`
	IsAdmin   bool   `json:"is_admin"`
}

// External API model (different field names)
type ExternalUser struct {
	UserID      int64  `api:"user_id"`
	EmailAddr   string `api:"email_address"`
	GivenName   string `api:"given_name"`
	FamilyName  string `api:"family_name"`
	DateOfBirth string `api:"dob"`
}

// Internal model using same tags
type InternalUser struct {
	ID        int64  `api:"user_id"`
	Email     string `api:"email_address"`
	FirstName string `api:"given_name"`
	LastName  string `api:"family_name"`
}

func main() {
	fmt.Println("=== gref Struct Mapper Example ===")
	fmt.Println()

	// Source entity
	entity := UserEntity{
		ID:           12345,
		Email:        "alice@example.com",
		PasswordHash: "bcrypt$2a$10$...",
		FirstName:    "Alice",
		LastName:     "Smith",
		DateOfBirth:  time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC),
		CreatedAt:    time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Now(),
		IsAdmin:      true,
		LoginCount:   42,
	}

	// 1. Simple auto-mapping (same field names)
	fmt.Println("1. Auto-mapping by field name:")
	type SimpleDTO struct {
		ID        int64
		Email     string
		FirstName string
		LastName  string
		IsAdmin   bool
	}
	var simpleDTO SimpleDTO
	AutoMap(&entity, &simpleDTO)
	fmt.Printf("   Source:  ID=%d, Email=%s, Name=%s %s\n",
		entity.ID, entity.Email, entity.FirstName, entity.LastName)
	fmt.Printf("   Mapped:  ID=%d, Email=%s, Name=%s %s\n",
		simpleDTO.ID, simpleDTO.Email, simpleDTO.FirstName, simpleDTO.LastName)
	fmt.Println()

	// 2. Mapping with computed fields (manual for complex transforms)
	fmt.Println("2. Mapping with computed fields:")
	dto := &UserDTO{
		ID:        entity.ID,
		Email:     entity.Email,
		FullName:  entity.FirstName + " " + entity.LastName,
		Age:       int(time.Since(entity.DateOfBirth).Hours() / 24 / 365),
		MemberFor: fmt.Sprintf("%.1f years", time.Since(entity.CreatedAt).Hours()/24/365),
		IsAdmin:   entity.IsAdmin,
	}
	fmt.Printf("   UserDTO: %+v\n", dto)
	fmt.Println()

	// 3. Explicit field mapping
	fmt.Println("3. Explicit field mapping:")
	type RenamedDTO struct {
		UserID    int64
		UserEmail string
		Admin     bool
	}
	var renamed RenamedDTO
	NewMapper().
		Map("ID", "UserID").
		Map("Email", "UserEmail").
		Map("IsAdmin", "Admin").
		Copy(&entity, &renamed)
	fmt.Printf("   Mapped: UserID=%d, UserEmail=%s, Admin=%v\n",
		renamed.UserID, renamed.UserEmail, renamed.Admin)
	fmt.Println()

	// 4. Case-insensitive mapping
	fmt.Println("4. Case-insensitive mapping:")
	type LowerDTO struct {
		Id        int64
		Email     string
		Firstname string
		Lastname  string
	}
	var lower LowerDTO
	NewMapper().IgnoreCase().Copy(&entity, &lower)
	fmt.Printf("   Mapped: Id=%d, Firstname=%s, Lastname=%s\n",
		lower.Id, lower.Firstname, lower.Lastname)
	fmt.Println()

	// 5. Tag-based mapping
	fmt.Println("5. Tag-based mapping (different struct types, same tags):")
	external := ExternalUser{
		UserID:      99,
		EmailAddr:   "bob@external.com",
		GivenName:   "Bob",
		FamilyName:  "Jones",
		DateOfBirth: "1985-03-20",
	}

	var internal InternalUser
	AutoMapTag(&external, &internal, "api")
	fmt.Printf("   External: UserID=%d, Email=%s, Name=%s %s\n",
		external.UserID, external.EmailAddr, external.GivenName, external.FamilyName)
	fmt.Printf("   Internal: ID=%d, Email=%s, Name=%s %s\n",
		internal.ID, internal.Email, internal.FirstName, internal.LastName)
	fmt.Println()

	// 6. Nested struct mapping
	fmt.Println("6. Nested struct mapping:")
	type SourceAddr struct {
		Street string
		City   string
		Zip    string
	}
	type SourcePerson struct {
		Name    string
		Address SourceAddr
	}
	type DestAddr struct {
		Street string
		City   string
		Zip    string // Same name for this example
	}
	type DestPerson struct {
		Name    string
		Address DestAddr
	}

	src := SourcePerson{
		Name: "Charlie",
		Address: SourceAddr{
			Street: "123 Main St",
			City:   "Seattle",
			Zip:    "98101",
		},
	}

	var dst DestPerson
	AutoMap(&src, &dst)
	fmt.Printf("   Source: %+v\n", src)
	fmt.Printf("   Dest:   %+v\n", dst)
	fmt.Println()

	// 7. Batch mapping
	fmt.Println("7. Batch mapping (slice of structs):")
	entities := []UserEntity{
		{ID: 1, Email: "a@example.com", FirstName: "Alice"},
		{ID: 2, Email: "b@example.com", FirstName: "Bob"},
		{ID: 3, Email: "c@example.com", FirstName: "Charlie"},
	}

	dtos := make([]SimpleDTO, len(entities))
	for i := range entities {
		AutoMap(&entities[i], &dtos[i])
	}
	for _, d := range dtos {
		fmt.Printf("   ID=%d, Email=%s, Name=%s\n", d.ID, d.Email, d.FirstName)
	}
}
