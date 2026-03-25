// Package main demonstrates using gref to generate fake/random test data
// based on struct field types and tags.
package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/vladimirvivien/gref"
)

// Faker generates fake data for structs
type Faker struct {
	rng *rand.Rand
}

// New creates a new Faker with optional seed
func New(seed ...int64) *Faker {
	var s int64
	if len(seed) > 0 {
		s = seed[0]
	} else {
		s = time.Now().UnixNano()
	}
	return &Faker{rng: rand.New(rand.NewSource(s))}
}

// Fill populates a struct with fake data based on field types and `fake` tags
func (f *Faker) Fill(v any) {
	s := gref.From(v).Struct()
	f.fillStruct(s)
}

func (f *Faker) fillStruct(s gref.Struct) {
	s.Fields().Exported().Each(func(field gref.Field) bool {
		if !field.CanSet() {
			return true
		}

		tag := field.ParsedTag("fake")

		// Handle nested structs
		if field.Kind() == gref.StructKind && !isSpecialType(field.Type()) {
			f.fillStruct(field.Struct())
			return true
		}

		// Generate value based on tag or type
		var value any
		if tag.Exists && tag.Name != "" {
			value = f.generateByTag(tag.Name, field.Kind())
		} else {
			value = f.generateByKind(field.Kind(), field.Type())
		}

		if value != nil {
			field.Set(value)
		}

		return true
	})
}

func (f *Faker) generateByTag(tagName string, kind gref.Kind) any {
	switch tagName {
	case "name":
		return f.name()
	case "first_name":
		return f.firstName()
	case "last_name":
		return f.lastName()
	case "email":
		return f.email()
	case "phone":
		return f.phone()
	case "url", "website":
		return f.url()
	case "username":
		return f.username()
	case "uuid":
		return f.uuid()
	case "sentence":
		return f.sentence()
	case "paragraph":
		return f.paragraph()
	case "address":
		return f.address()
	case "city":
		return f.city()
	case "country":
		return f.country()
	case "zip":
		return f.zip()
	case "company":
		return f.company()
	case "job_title":
		return f.jobTitle()
	case "date":
		return f.date()
	case "past_date":
		return f.pastDate()
	case "future_date":
		return f.futureDate()
	case "skip", "-":
		return nil
	default:
		return f.generateByKind(kind, nil)
	}
}

func (f *Faker) generateByKind(kind gref.Kind, t gref.Type) any {
	switch kind {
	case gref.String:
		return f.word()
	case gref.Int, gref.Int8, gref.Int16, gref.Int32, gref.Int64:
		return f.rng.Intn(1000)
	case gref.Uint, gref.Uint8, gref.Uint16, gref.Uint32, gref.Uint64:
		return uint(f.rng.Intn(1000))
	case gref.Float32, gref.Float64:
		return f.rng.Float64() * 1000
	case gref.Bool:
		return f.rng.Intn(2) == 1
	case gref.SliceKind:
		if t != nil {
			return f.generateSlice(t)
		}
		return nil
	default:
		if t != nil && t == gref.TypeOf[time.Time]() {
			return f.date()
		}
		return nil
	}
}

func (f *Faker) generateSlice(t gref.Type) any {
	length := f.rng.Intn(5) + 1 // 1-5 elements

	// Create slice using gref
	slice := gref.Def().Slice(t.Elem(), length, length)

	for i := 0; i < length; i++ {
		elemKind := t.Elem().Kind()
		val := f.generateByKind(elemKind, t.Elem())
		if val != nil {
			slice.Set(i, val)
		}
	}
	return slice.Interface()
}

func isSpecialType(t gref.Type) bool {
	return t == gref.TypeOf[time.Time]()
}

// --- Data generators ---

var firstNames = []string{"Alice", "Bob", "Charlie", "Diana", "Eve", "Frank", "Grace", "Henry", "Ivy", "Jack"}
var lastNames = []string{"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis", "Wilson", "Moore"}
var domains = []string{"example.com", "test.org", "demo.io", "sample.net", "mock.dev"}
var companies = []string{"Acme Corp", "Globex", "Initech", "Umbrella", "Stark Industries", "Wayne Enterprises"}
var jobTitles = []string{"Engineer", "Manager", "Designer", "Analyst", "Developer", "Architect", "Consultant"}
var cities = []string{"New York", "Los Angeles", "Chicago", "Houston", "Phoenix", "Seattle", "Boston", "Denver"}
var countries = []string{"USA", "Canada", "UK", "Germany", "France", "Japan", "Australia", "Brazil"}
var words = []string{"lorem", "ipsum", "dolor", "sit", "amet", "consectetur", "adipiscing", "elit", "sed", "do"}

func (f *Faker) pick(items []string) string {
	return items[f.rng.Intn(len(items))]
}

func (f *Faker) firstName() string { return f.pick(firstNames) }
func (f *Faker) lastName() string  { return f.pick(lastNames) }
func (f *Faker) name() string      { return f.firstName() + " " + f.lastName() }
func (f *Faker) word() string      { return f.pick(words) }
func (f *Faker) city() string      { return f.pick(cities) }
func (f *Faker) country() string   { return f.pick(countries) }
func (f *Faker) company() string   { return f.pick(companies) }
func (f *Faker) jobTitle() string  { return f.pick(jobTitles) }

func (f *Faker) email() string {
	return strings.ToLower(f.firstName()) + "." + strings.ToLower(f.lastName()) + "@" + f.pick(domains)
}

func (f *Faker) username() string {
	return strings.ToLower(f.firstName()) + fmt.Sprintf("%d", f.rng.Intn(1000))
}

func (f *Faker) phone() string {
	return fmt.Sprintf("+1-%03d-%03d-%04d", f.rng.Intn(1000), f.rng.Intn(1000), f.rng.Intn(10000))
}

func (f *Faker) url() string {
	return "https://" + f.pick(domains) + "/" + f.word()
}

func (f *Faker) uuid() string {
	b := make([]byte, 16)
	f.rng.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (f *Faker) address() string {
	return fmt.Sprintf("%d %s St", f.rng.Intn(9999)+1, f.pick([]string{"Main", "Oak", "Elm", "Park", "Lake", "Hill"}))
}

func (f *Faker) zip() string {
	return fmt.Sprintf("%05d", f.rng.Intn(100000))
}

func (f *Faker) sentence() string {
	length := f.rng.Intn(8) + 3 // 3-10 words
	wordList := make([]string, length)
	for i := range wordList {
		wordList[i] = f.word()
	}
	wordList[0] = strings.ToUpper(wordList[0][:1]) + wordList[0][1:]
	return strings.Join(wordList, " ") + "."
}

func (f *Faker) paragraph() string {
	sentences := f.rng.Intn(3) + 2 // 2-4 sentences
	parts := make([]string, sentences)
	for i := range parts {
		parts[i] = f.sentence()
	}
	return strings.Join(parts, " ")
}

func (f *Faker) date() time.Time {
	// Random date within last 2 years
	days := f.rng.Intn(730)
	return time.Now().AddDate(0, 0, -days)
}

func (f *Faker) pastDate() time.Time {
	days := f.rng.Intn(365) + 1
	return time.Now().AddDate(0, 0, -days)
}

func (f *Faker) futureDate() time.Time {
	days := f.rng.Intn(365) + 1
	return time.Now().AddDate(0, 0, days)
}

// --- Example structs ---

type User struct {
	ID        int       `fake:"-"` // Skip - we'll set manually
	Username  string    `fake:"username"`
	Email     string    `fake:"email"`
	FirstName string    `fake:"first_name"`
	LastName  string    `fake:"last_name"`
	Phone     string    `fake:"phone"`
	Bio       string    `fake:"paragraph"`
	Website   string    `fake:"url"`
	CreatedAt time.Time `fake:"past_date"`
	IsActive  bool
}

type Company struct {
	ID      int    `fake:"-"`
	Name    string `fake:"company"`
	Website string `fake:"website"`
	Address Address
}

type Address struct {
	Street  string `fake:"address"`
	City    string `fake:"city"`
	Country string `fake:"country"`
	ZipCode string `fake:"zip"`
}

type Order struct {
	ID         string    `fake:"uuid"`
	CustomerID int
	Items      []string  `fake:"word"` // Will generate slice of words
	Total      float64
	PlacedAt   time.Time `fake:"past_date"`
}

func main() {
	fmt.Println("=== gref Faker Example ===")
	fmt.Println()

	// Use fixed seed for reproducible output
	faker := New(42)

	// Generate fake users
	fmt.Println("1. Generate fake users:")
	for i := 1; i <= 3; i++ {
		user := &User{ID: i}
		faker.Fill(user)
		fmt.Printf("   User %d:\n", i)
		fmt.Printf("     Username:  %s\n", user.Username)
		fmt.Printf("     Email:     %s\n", user.Email)
		fmt.Printf("     Name:      %s %s\n", user.FirstName, user.LastName)
		fmt.Printf("     Phone:     %s\n", user.Phone)
		fmt.Printf("     Website:   %s\n", user.Website)
		fmt.Printf("     Active:    %v\n", user.IsActive)
		fmt.Printf("     Created:   %s\n", user.CreatedAt.Format("2006-01-02"))
		fmt.Printf("     Bio:       %.50s...\n", user.Bio)
		fmt.Println()
	}

	// Generate company with nested address
	fmt.Println("2. Generate company with nested struct:")
	company := &Company{ID: 1}
	faker.Fill(company)
	fmt.Printf("   Company: %s\n", company.Name)
	fmt.Printf("   Website: %s\n", company.Website)
	fmt.Printf("   Address:\n")
	fmt.Printf("     Street:  %s\n", company.Address.Street)
	fmt.Printf("     City:    %s\n", company.Address.City)
	fmt.Printf("     Country: %s\n", company.Address.Country)
	fmt.Printf("     Zip:     %s\n", company.Address.ZipCode)
	fmt.Println()

	// Generate orders with slices
	fmt.Println("3. Generate orders with slices:")
	for i := 1; i <= 2; i++ {
		order := &Order{}
		faker.Fill(order)
		fmt.Printf("   Order: %s\n", order.ID)
		fmt.Printf("     Customer: %d\n", order.CustomerID)
		fmt.Printf("     Items:    %v\n", order.Items)
		fmt.Printf("     Total:    $%.2f\n", order.Total)
		fmt.Printf("     Placed:   %s\n", order.PlacedAt.Format("2006-01-02"))
		fmt.Println()
	}

	// Batch generation helper
	fmt.Println("4. Batch generation (5 addresses):")
	addresses := make([]Address, 5)
	for i := range addresses {
		faker.Fill(&addresses[i])
		fmt.Printf("   %d. %s, %s, %s %s\n",
			i+1, addresses[i].Street, addresses[i].City, addresses[i].Country, addresses[i].ZipCode)
	}
	fmt.Println()

	// Show how fields without tags get type-based values
	fmt.Println("5. Type-based generation (no tags):")
	type Simple struct {
		Name    string
		Count   int
		Price   float64
		Enabled bool
		Tags    []string
	}
	simple := &Simple{}
	faker.Fill(simple)
	fmt.Printf("   Name:    %s\n", simple.Name)
	fmt.Printf("   Count:   %d\n", simple.Count)
	fmt.Printf("   Price:   %.2f\n", simple.Price)
	fmt.Printf("   Enabled: %v\n", simple.Enabled)
	fmt.Printf("   Tags:    %v\n", simple.Tags)
}
