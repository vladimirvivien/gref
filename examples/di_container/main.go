// Package main demonstrates using gref to build a simple dependency injection container.
// It supports constructor injection and struct field injection via tags.
package main

import (
	"fmt"

	"github.com/vladimirvivien/gref"
)

// Container is a simple dependency injection container
type Container struct {
	services map[gref.Type]any
}

// NewContainer creates a new DI container
func NewContainer() *Container {
	return &Container{
		services: make(map[gref.Type]any),
	}
}

// Register registers a service instance
func (c *Container) Register(service any) {
	t := gref.From(service).Type()
	c.services[t] = service
	fmt.Printf("[DI] Registered: %s\n", t)
}

// RegisterAs registers a service as a specific interface type
func (c *Container) RegisterAs(service any, asType any) {
	t := gref.From(asType).Type().Elem()
	c.services[t] = service
	fmt.Printf("[DI] Registered %T as %s\n", service, t)
}

// Resolve creates an instance with dependencies injected
func (c *Container) Resolve(target any) error {
	s := gref.From(target).Struct()

	s.Fields().WithTag("inject").Each(func(field gref.Field) bool {
		// Find service by type
		fieldType := field.Type()
		service, ok := c.services[fieldType]
		if !ok {
			// Try to find by interface
			for t, svc := range c.services {
				if t.Implements(fieldType) || gref.PtrTo(t).Implements(fieldType) {
					service = svc
					ok = true
					break
				}
			}
		}

		if ok {
			field.Set(service)
			fmt.Printf("[DI] Injected %s into %s\n", fieldType, field.Name())
		}

		return true
	})

	return nil
}

// ResolveFunc calls a function with dependencies resolved from the container
func (c *Container) ResolveFunc(fn any) gref.Results {
	f := gref.From(fn).Func()

	// Build arguments from container
	args := make([]any, f.NumIn())
	for i := 0; i < f.NumIn(); i++ {
		paramType := f.In(i)
		if service, ok := c.services[paramType]; ok {
			args[i] = service
		} else {
			// Try interface matching
			for _, svc := range c.services {
				svcType := gref.From(svc).Type()
				if svcType.AssignableTo(paramType) {
					args[i] = svc
					break
				}
			}
		}
	}

	return f.Call(args...)
}

// --- Example Services ---

type Logger interface {
	Log(msg string)
}

type ConsoleLogger struct {
	Prefix string
}

func (l *ConsoleLogger) Log(msg string) {
	fmt.Printf("%s: %s\n", l.Prefix, msg)
}

type Config struct {
	DatabaseURL string
	APIKey      string
}

type Database struct {
	URL    string
	Logger Logger `inject:"logger"`
}

func (d *Database) Connect() {
	d.Logger.Log(fmt.Sprintf("Connecting to %s", d.URL))
}

func (d *Database) Query(sql string) {
	d.Logger.Log(fmt.Sprintf("Executing: %s", sql))
}

// UserService depends on Database and Logger
type UserService struct {
	DB     *Database `inject:"database"`
	Logger Logger    `inject:"logger"`
	Config *Config   `inject:"config"`
}

func (s *UserService) GetUser(id int) string {
	s.Logger.Log(fmt.Sprintf("Getting user %d", id))
	s.DB.Query(fmt.Sprintf("SELECT * FROM users WHERE id = %d", id))
	return fmt.Sprintf("User-%d", id)
}

// --- Constructor function example ---

func NewUserHandler(logger Logger, db *Database) string {
	logger.Log("Creating UserHandler")
	db.Query("SELECT 1")
	return "UserHandler initialized"
}

func main() {
	fmt.Println("=== gref Dependency Injection Example ===")
	fmt.Println()

	// Create container and register services
	container := NewContainer()

	// Register concrete implementations
	logger := &ConsoleLogger{Prefix: "[APP]"}
	container.RegisterAs(logger, (*Logger)(nil))

	config := &Config{
		DatabaseURL: "postgres://localhost:5432/mydb",
		APIKey:      "secret-key-123",
	}
	container.Register(config)

	// Create and inject Database
	db := &Database{URL: config.DatabaseURL}
	container.Resolve(db)
	container.Register(db)

	fmt.Println()

	// Resolve a service with multiple dependencies
	fmt.Println("Resolving UserService with dependencies:")
	userService := &UserService{}
	container.Resolve(userService)

	fmt.Println()

	// Use the resolved service
	fmt.Println("Using resolved service:")
	user := userService.GetUser(42)
	fmt.Printf("Result: %s\n", user)

	fmt.Println()

	// Constructor injection - resolve function parameters
	fmt.Println("Constructor injection via function:")
	result := container.ResolveFunc(NewUserHandler)
	fmt.Printf("Result: %v\n", result.First().Interface())

	fmt.Println()

	// Show service registry
	fmt.Println("Registered services:")
	for t := range container.services {
		fmt.Printf("  - %s\n", t)
	}
}
