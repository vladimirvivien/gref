// Package main demonstrates using gref to build a type-based event router.
// Events are routed to handlers based on their type, with support for
// middleware, wildcards, and extracting event data.
package main

import (
	"fmt"
	"time"

	"github.com/vladimirvivien/gref"
)

// Event is the base interface for all events
type Event interface {
	EventName() string
	Timestamp() time.Time
}

// BaseEvent provides common event fields
type BaseEvent struct {
	Time time.Time
}

func (e BaseEvent) Timestamp() time.Time { return e.Time }

// --- Concrete Events ---

type UserCreated struct {
	BaseEvent
	UserID   int
	Username string
	Email    string
}

func (e UserCreated) EventName() string { return "user.created" }

type UserUpdated struct {
	BaseEvent
	UserID  int
	Changes map[string]any
}

func (e UserUpdated) EventName() string { return "user.updated" }

type UserDeleted struct {
	BaseEvent
	UserID int
	Reason string
}

func (e UserDeleted) EventName() string { return "user.deleted" }

type OrderPlaced struct {
	BaseEvent
	OrderID    string
	CustomerID int
	Amount     float64
	Items      []string
}

func (e OrderPlaced) EventName() string { return "order.placed" }

// EventRouter routes events to registered handlers
type EventRouter struct {
	handlers   map[gref.Type][]any
	middleware []func(Event, func())
}

// NewEventRouter creates a new router
func NewEventRouter() *EventRouter {
	return &EventRouter{
		handlers: make(map[gref.Type][]any),
	}
}

// On registers a handler for a specific event type
// Handler should be func(EventType) or func(EventType) error
func (r *EventRouter) On(handler any) {
	f := gref.From(handler).Func()
	if f.NumIn() != 1 {
		panic("handler must accept exactly one argument")
	}

	eventType := f.In(0)
	r.handlers[eventType] = append(r.handlers[eventType], handler)
	fmt.Printf("[Router] Registered handler for %s\n", eventType.Name())
}

// Use adds middleware that runs before handlers
func (r *EventRouter) Use(mw func(Event, func())) {
	r.middleware = append(r.middleware, mw)
}

// Dispatch sends an event to all registered handlers
func (r *EventRouter) Dispatch(event Event) error {
	eventType := gref.From(event).Type()
	handlers := r.handlers[eventType]

	if len(handlers) == 0 {
		fmt.Printf("[Router] No handlers for %s\n", eventType.Name())
		return nil
	}

	fmt.Printf("[Router] Dispatching %s to %d handler(s)\n", event.EventName(), len(handlers))

	// Build the handler chain
	runHandlers := func() {
		for _, h := range handlers {
			f := gref.From(h).Func()
			results := f.Call(event)

			// Check for error return
			if results.Len() > 0 {
				if err := results.Error(); err != nil {
					fmt.Printf("[Router] Handler error: %v\n", err)
				}
			}
		}
	}

	// Apply middleware in reverse order
	chain := runHandlers
	for i := len(r.middleware) - 1; i >= 0; i-- {
		mw := r.middleware[i]
		next := chain
		chain = func() { mw(event, next) }
	}

	chain()
	return nil
}

// ExtractEventData extracts all public fields from an event into a map
func ExtractEventData(event Event) map[string]any {
	s := gref.From(event).Struct()
	data := make(map[string]any)

	s.Fields().Exported().Each(func(field gref.Field) bool {
		name := field.Name()

		// Skip embedded BaseEvent fields by checking if it's the Time field
		if name == "BaseEvent" || name == "Time" {
			return true
		}

		data[name] = field.Interface()
		return true
	})

	// Add event metadata
	data["_event"] = event.EventName()
	data["_timestamp"] = event.Timestamp()

	return data
}

// EventLogger is middleware that logs all events
func EventLogger(event Event, next func()) {
	fmt.Printf("[LOG] Event: %s at %s\n", event.EventName(), event.Timestamp().Format(time.RFC3339))
	data := ExtractEventData(event)
	for k, v := range data {
		if k[0] != '_' {
			fmt.Printf("[LOG]   %s: %v\n", k, v)
		}
	}
	next()
}

// EventTimer is middleware that times event processing
func EventTimer(event Event, next func()) {
	start := time.Now()
	next()
	fmt.Printf("[TIMER] %s processed in %v\n", event.EventName(), time.Since(start))
}

func main() {
	fmt.Println("=== gref Event Router Example ===")
	fmt.Println()

	router := NewEventRouter()

	// Add middleware
	router.Use(EventLogger)
	router.Use(EventTimer)

	fmt.Println()

	// Register handlers for different event types
	router.On(func(e UserCreated) {
		fmt.Printf("[Handler] Welcome new user: %s (ID: %d)\n", e.Username, e.UserID)
		// Simulate sending welcome email
		time.Sleep(10 * time.Millisecond)
	})

	router.On(func(e UserCreated) {
		fmt.Printf("[Handler] Creating default settings for user %d\n", e.UserID)
	})

	router.On(func(e UserUpdated) {
		fmt.Printf("[Handler] User %d updated: %v\n", e.UserID, e.Changes)
	})

	router.On(func(e UserDeleted) error {
		fmt.Printf("[Handler] Cleaning up data for user %d (reason: %s)\n", e.UserID, e.Reason)
		if e.Reason == "fraud" {
			return fmt.Errorf("fraud case requires manual review")
		}
		return nil
	})

	router.On(func(e OrderPlaced) {
		fmt.Printf("[Handler] Processing order %s for customer %d: $%.2f\n",
			e.OrderID, e.CustomerID, e.Amount)
		fmt.Printf("[Handler]   Items: %v\n", e.Items)
	})

	fmt.Println()
	fmt.Println("--- Dispatching Events ---")
	fmt.Println()

	// Dispatch various events
	router.Dispatch(UserCreated{
		BaseEvent: BaseEvent{Time: time.Now()},
		UserID:    1,
		Username:  "alice",
		Email:     "alice@example.com",
	})

	fmt.Println()

	router.Dispatch(UserUpdated{
		BaseEvent: BaseEvent{Time: time.Now()},
		UserID:    1,
		Changes:   map[string]any{"email": "alice.new@example.com"},
	})

	fmt.Println()

	router.Dispatch(OrderPlaced{
		BaseEvent:  BaseEvent{Time: time.Now()},
		OrderID:    "ORD-12345",
		CustomerID: 1,
		Amount:     99.99,
		Items:      []string{"Widget", "Gadget", "Gizmo"},
	})

	fmt.Println()

	router.Dispatch(UserDeleted{
		BaseEvent: BaseEvent{Time: time.Now()},
		UserID:    2,
		Reason:    "requested",
	})

	fmt.Println()

	// Demonstrate event with no handlers
	router.Dispatch(UnknownEvent{
		BaseEvent: BaseEvent{Time: time.Now()},
		Data:      "test",
	})

	fmt.Println()

	// Show event data extraction
	fmt.Println("--- Event Data Extraction ---")
	event := OrderPlaced{
		BaseEvent:  BaseEvent{Time: time.Now()},
		OrderID:    "ORD-99999",
		CustomerID: 42,
		Amount:     199.50,
		Items:      []string{"Premium Widget"},
	}
	data := ExtractEventData(event)
	fmt.Println("Extracted event data:")
	for k, v := range data {
		fmt.Printf("  %s: %v\n", k, v)
	}
}

// UnknownEvent for testing no-handler case
type UnknownEvent struct {
	BaseEvent
	Data string
}

func (e UnknownEvent) EventName() string { return "unknown" }
