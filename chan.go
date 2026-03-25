package gref

import (
	"reflect"
)

// Chan provides operations on channel values.
// Create via Value.Chan() or MakeChan[T]().
//
// Example:
//
//	c := refl.From(make(chan int, 3)).Chan()
//	c.Send(42)
//	val, ok := c.Recv()
//
//	// Or create dynamically
//	c := refl.MakeChan[string](5)
//	c.Send("hello")
type Chan struct {
	rv       reflect.Value
	elemType reflect.Type
	dir      ChanDir
}

// reflectValue implements Valuable.
func (c Chan) reflectValue() reflect.Value {
	return c.rv
}

// --- Size Information ---

// Len returns the number of elements in the buffer.
func (c Chan) Len() int {
	return c.rv.Len()
}

// Cap returns the channel capacity.
func (c Chan) Cap() int {
	return c.rv.Cap()
}

// IsNil returns true if the channel is nil.
func (c Chan) IsNil() bool {
	return c.rv.IsNil()
}

// IsFull returns true if the buffer is full.
func (c Chan) IsFull() bool {
	return c.rv.Len() == c.rv.Cap()
}

// IsEmpty returns true if the buffer is empty.
func (c Chan) IsEmpty() bool {
	return c.rv.Len() == 0
}

// --- Type Information ---

// ElemType returns the element type.
func (c Chan) ElemType() reflect.Type {
	return c.elemType
}

// Dir returns the channel direction.
func (c Chan) Dir() ChanDir {
	return c.dir
}

// Type returns the channel type.
func (c Chan) Type() reflect.Type {
	return c.rv.Type()
}

// --- Direction Checks ---

// CanSend returns true if the channel allows sending.
func (c Chan) CanSend() bool {
	return c.dir == SendDir || c.dir == BothDir
}

// CanRecv returns true if the channel allows receiving.
func (c Chan) CanRecv() bool {
	return c.dir == RecvDir || c.dir == BothDir
}

// IsSendOnly returns true if send-only.
func (c Chan) IsSendOnly() bool {
	return c.dir == SendDir
}

// IsRecvOnly returns true if receive-only.
func (c Chan) IsRecvOnly() bool {
	return c.dir == RecvDir
}

// IsBidirectional returns true if bidirectional.
func (c Chan) IsBidirectional() bool {
	return c.dir == BothDir
}

// --- Send Operations ---

// Send sends a value (blocking). Returns the channel for chaining.
// Panics if not a send channel.
//
// Example:
//
//	c.Send(1).Send(2).Send(3)
func (c Chan) Send(value any) Chan {
	if !c.CanSend() {
		panic("gref: cannot send on receive-only channel")
	}
	c.rv.Send(reflect.ValueOf(value))
	return c
}

// TrySend attempts to send without blocking. Returns true if sent.
func (c Chan) TrySend(value any) bool {
	if !c.CanSend() {
		panic("gref: cannot send on receive-only channel")
	}
	return c.rv.TrySend(reflect.ValueOf(value))
}

// SendAll sends multiple values.
func (c Chan) SendAll(values ...any) Chan {
	for _, v := range values {
		c.Send(v)
	}
	return c
}

// --- Receive Operations ---

// Recv receives a value (blocking).
// Returns (value, open). open is false if channel is closed.
// Panics if not a receive channel.
//
// Example:
//
//	val, ok := c.Recv()
//	if ok {
//	    n, _ := refl.Get[int](val)
//	}
func (c Chan) Recv() (Value, bool) {
	if !c.CanRecv() {
		panic("gref: cannot receive on send-only channel")
	}
	val, ok := c.rv.Recv()
	return Value{rv: val}, ok
}

// TryRecv attempts to receive without blocking.
// Returns (value, received, open).
//   - received: true if a value was received
//   - open: true if channel is still open
func (c Chan) TryRecv() (Value, bool, bool) {
	if !c.CanRecv() {
		panic("gref: cannot receive on send-only channel")
	}
	val, ok := c.rv.TryRecv()
	if !val.IsValid() {
		return Value{}, false, true // Would block
	}
	if !ok {
		return Value{}, false, false // Closed
	}
	return Value{rv: val}, true, true
}

// --- Close ---

// Close closes the channel. Panics if receive-only.
func (c Chan) Close() {
	if c.IsRecvOnly() {
		panic("gref: cannot close receive-only channel")
	}
	c.rv.Close()
}

// --- Iteration ---

// Range iterates until channel is closed.
// fn receives each value. Return false to stop early.
//
// Example:
//
//	c.Range(func(v refl.Value) bool {
//	    n, _ := refl.Get[int](v)
//	    fmt.Println(n)
//	    return true  // continue
//	})
func (c Chan) Range(fn func(Value) bool) {
	if !c.CanRecv() {
		panic("gref: cannot range over send-only channel")
	}
	for {
		val, ok := c.rv.Recv()
		if !ok {
			break
		}
		if !fn(Value{rv: val}) {
			break
		}
	}
}

// Drain receives all buffered values without blocking.
func (c Chan) Drain() []Value {
	if !c.CanRecv() {
		panic("gref: cannot drain send-only channel")
	}

	var values []Value
	for {
		val, ok := c.rv.TryRecv()
		if !val.IsValid() {
			break // Would block
		}
		if !ok {
			break // Closed
		}
		values = append(values, Value{rv: val})
	}
	return values
}

// --- Conversion ---

// Interface returns the channel as any.
func (c Chan) Interface() any {
	return c.rv.Interface()
}

// Value returns this channel as a Value.
func (c Chan) Value() Value {
	return Value{rv: c.rv}
}

// ============================================================================
// Select Operations
// ============================================================================

// SelectCase represents a case for Select operations.
type SelectCase struct {
	Dir     SelectDir
	Chan    Chan
	SendVal any // For send cases
}

// SelectDir indicates the direction of a select case.
type SelectDir int

const (
	SelectSend SelectDir = iota + 1
	SelectRecv
	SelectDefault
)

// Select performs a reflect.Select operation.
// Returns (chosen index, received value, channel open).
//
// Example:
//
//	chosen, val, ok := refl.Select([]refl.SelectCase{
//	    {Dir: refl.SelectRecv, Chan: c1},
//	    {Dir: refl.SelectRecv, Chan: c2},
//	    {Dir: refl.SelectDefault},
//	})
func Select(cases []SelectCase) (int, Value, bool) {
	selectCases := make([]reflect.SelectCase, len(cases))

	for i, c := range cases {
		switch c.Dir {
		case SelectRecv:
			selectCases[i] = reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: c.Chan.rv,
			}
		case SelectSend:
			selectCases[i] = reflect.SelectCase{
				Dir:  reflect.SelectSend,
				Chan: c.Chan.rv,
				Send: reflect.ValueOf(c.SendVal),
			}
		case SelectDefault:
			selectCases[i] = reflect.SelectCase{
				Dir: reflect.SelectDefault,
			}
		}
	}

	chosen, recv, recvOK := reflect.Select(selectCases)

	if !recv.IsValid() {
		return chosen, Value{}, recvOK
	}

	return chosen, Value{rv: recv}, recvOK
}
