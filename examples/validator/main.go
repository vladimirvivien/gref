// Package main demonstrates using gref for struct validation
// based on custom struct tags.
package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/vladimirvivien/gref"
)

// ValidationError represents a single validation failure
type ValidationError struct {
	Field   string
	Rule    string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// Validate validates a struct based on `validate` tags.
// Supported rules:
//   - required: field must not be zero value
//   - min=N: string length or numeric value >= N
//   - max=N: string length or numeric value <= N
//   - email: must be valid email format
//   - oneof=a|b|c: value must be one of the listed options
func Validate(v any) ValidationErrors {
	var errs ValidationErrors
	s := gref.From(v).Struct()

	s.Fields().Each(func(field gref.Field) bool {
		tag := field.ParsedTag("validate")
		if !tag.Exists {
			return true
		}

		fieldErrs := validateField(field, tag)
		errs = append(errs, fieldErrs...)
		return true
	})

	return errs
}

func validateField(field gref.Field, tag gref.ParsedTag) []ValidationError {
	var errs []ValidationError

	// Collect all rules (tag.Name is the first rule, Options contains additional ones)
	rules := []string{}
	if tag.Name != "" {
		rules = append(rules, tag.Name)
	}
	rules = append(rules, tag.Options...)

	for _, rule := range rules {
		if err := applyRule(field, rule); err != nil {
			errs = append(errs, *err)
		}
	}

	return errs
}

func applyRule(field gref.Field, rule string) *ValidationError {
	// Parse rule with optional parameter: "min=5" -> ("min", "5")
	ruleName, ruleParam := parseRule(rule)

	switch ruleName {
	case "required":
		if field.IsZero() {
			return &ValidationError{
				Field:   field.Name(),
				Rule:    "required",
				Message: "is required",
			}
		}

	case "min":
		minVal, _ := strconv.Atoi(ruleParam)
		if err := checkMin(field, minVal); err != nil {
			return &ValidationError{
				Field:   field.Name(),
				Rule:    rule,
				Message: err.Error(),
			}
		}

	case "max":
		maxVal, _ := strconv.Atoi(ruleParam)
		if err := checkMax(field, maxVal); err != nil {
			return &ValidationError{
				Field:   field.Name(),
				Rule:    rule,
				Message: err.Error(),
			}
		}

	case "email":
		if result := gref.TryGet[string](field); result.Ok() {
			str := result.OrZero()
			if str != "" && !isValidEmail(str) {
				return &ValidationError{
					Field:   field.Name(),
					Rule:    "email",
					Message: "must be a valid email address",
				}
			}
		}

	case "oneof":
		options := strings.Split(ruleParam, "|")
		if result := gref.TryGet[string](field); result.Ok() {
			str := result.OrZero()
			found := false
			for _, opt := range options {
				if str == opt {
					found = true
					break
				}
			}
			if !found {
				return &ValidationError{
					Field:   field.Name(),
					Rule:    rule,
					Message: fmt.Sprintf("must be one of: %s", strings.Join(options, ", ")),
				}
			}
		}
	}

	return nil
}

func parseRule(rule string) (name, param string) {
	if idx := strings.Index(rule, "="); idx != -1 {
		return rule[:idx], rule[idx+1:]
	}
	return rule, ""
}

func checkMin(field gref.Field, minVal int) error {
	switch field.Kind() {
	case gref.String:
		if result := gref.TryGet[string](field); result.Ok() {
			if len(result.OrZero()) < minVal {
				return fmt.Errorf("must be at least %d characters", minVal)
			}
		}
	case gref.Int:
		if result := gref.TryGet[int](field); result.Ok() {
			if result.OrZero() < minVal {
				return fmt.Errorf("must be at least %d", minVal)
			}
		}
	}
	return nil
}

func checkMax(field gref.Field, maxVal int) error {
	switch field.Kind() {
	case gref.String:
		if result := gref.TryGet[string](field); result.Ok() {
			if len(result.OrZero()) > maxVal {
				return fmt.Errorf("must be at most %d characters", maxVal)
			}
		}
	case gref.Int:
		if result := gref.TryGet[int](field); result.Ok() {
			if result.OrZero() > maxVal {
				return fmt.Errorf("must be at most %d", maxVal)
			}
		}
	}
	return nil
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

// Example structs for validation

type User struct {
	Username string `validate:"required,min=3,max=20"`
	Email    string `validate:"required,email"`
	Password string `validate:"required,min=8"`
	Age      int    `validate:"min=18,max=120"`
	Role     string `validate:"required,oneof=admin|user|guest"`
}

type Address struct {
	Street  string `validate:"required"`
	City    string `validate:"required,min=2"`
	ZipCode string `validate:"required,min=5,max=10"`
	Country string `validate:"required,oneof=US|CA|UK|DE|FR"`
}

func main() {
	fmt.Println("=== gref Validator Example ===")
	fmt.Println()

	// Test case 1: Valid user
	fmt.Println("Test 1: Valid user")
	validUser := User{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "securepass123",
		Age:      25,
		Role:     "admin",
	}

	if errs := Validate(&validUser); len(errs) > 0 {
		fmt.Printf("  Validation failed: %v\n", errs)
	} else {
		fmt.Println("  Validation passed!")
	}
	fmt.Println()

	// Test case 2: Invalid user - multiple violations
	fmt.Println("Test 2: Invalid user (multiple violations)")
	invalidUser := User{
		Username: "ab",           // too short (min=3)
		Email:    "not-an-email", // invalid email
		Password: "short",        // too short (min=8)
		Age:      15,             // too young (min=18)
		Role:     "superadmin",   // not in oneof list
	}

	if errs := Validate(&invalidUser); len(errs) > 0 {
		fmt.Println("  Validation errors:")
		for _, err := range errs {
			fmt.Printf("    - %s [%s]: %s\n", err.Field, err.Rule, err.Message)
		}
	}
	fmt.Println()

	// Test case 3: Missing required fields
	fmt.Println("Test 3: Missing required fields")
	emptyUser := User{
		Age: 30, // Only age is set
	}

	if errs := Validate(&emptyUser); len(errs) > 0 {
		fmt.Println("  Validation errors:")
		for _, err := range errs {
			fmt.Printf("    - %s [%s]: %s\n", err.Field, err.Rule, err.Message)
		}
	}
	fmt.Println()

	// Test case 4: Valid address
	fmt.Println("Test 4: Valid address")
	validAddress := Address{
		Street:  "123 Main St",
		City:    "New York",
		ZipCode: "10001",
		Country: "US",
	}

	if errs := Validate(&validAddress); len(errs) > 0 {
		fmt.Printf("  Validation failed: %v\n", errs)
	} else {
		fmt.Println("  Validation passed!")
	}
	fmt.Println()

	// Test case 5: Invalid address
	fmt.Println("Test 5: Invalid address")
	invalidAddress := Address{
		Street:  "",          // required
		City:    "X",         // too short
		ZipCode: "123",       // too short
		Country: "Australia", // not in oneof
	}

	if errs := Validate(&invalidAddress); len(errs) > 0 {
		fmt.Println("  Validation errors:")
		for _, err := range errs {
			fmt.Printf("    - %s [%s]: %s\n", err.Field, err.Rule, err.Message)
		}
	}
	fmt.Println()

	// Demonstrate programmatic field inspection
	fmt.Println("Field validation rules inspection:")
	s := gref.From(&User{}).Struct()
	s.Fields().Each(func(field gref.Field) bool {
		tag := field.ParsedTag("validate")
		if tag.Exists {
			rules := []string{}
			if tag.Name != "" {
				rules = append(rules, tag.Name)
			}
			rules = append(rules, tag.Options...)
			fmt.Printf("  %s: %v\n", field.Name(), rules)
		}
		return true
	})
}
