package gref

import (
	"fmt"
	"reflect"
)

// ============================================================================
// Deep Equality
// ============================================================================

// Equal compares two Valuables for deep equality.
func Equal(a, b Valuable) bool {
	return reflect.DeepEqual(
		a.reflectValue().Interface(),
		b.reflectValue().Interface(),
	)
}

// ============================================================================
// Deep Copy
// ============================================================================

// DeepCopy creates a deep copy of a value.
func DeepCopy(v any) any {
	if v == nil {
		return nil
	}
	rv := reflect.ValueOf(v)
	copied := deepCopyValue(rv)
	if !copied.IsValid() {
		return nil
	}
	return copied.Interface()
}

// DeepCopyInto copies src into dst (dst must be a pointer).
func DeepCopyInto(src, dst any) error {
	if dst == nil {
		return ErrNilValue
	}

	dstVal := reflect.ValueOf(dst)
	if dstVal.Kind() != reflect.Pointer || dstVal.IsNil() {
		return fmt.Errorf("%w: dst must be non-nil pointer", ErrTypeMismatch)
	}

	copied := deepCopyValue(reflect.ValueOf(src))
	if !copied.IsValid() {
		dstVal.Elem().Set(reflect.Zero(dstVal.Elem().Type()))
		return nil
	}

	if !copied.Type().AssignableTo(dstVal.Elem().Type()) {
		return fmt.Errorf("%w: cannot assign %s to %s", ErrTypeMismatch, copied.Type(), dstVal.Elem().Type())
	}

	dstVal.Elem().Set(copied)
	return nil
}

func deepCopyValue(rv reflect.Value) reflect.Value {
	if !rv.IsValid() {
		return rv
	}

	switch rv.Kind() {
	case reflect.Pointer:
		if rv.IsNil() {
			return reflect.Zero(rv.Type())
		}
		newPtr := reflect.New(rv.Type().Elem())
		newPtr.Elem().Set(deepCopyValue(rv.Elem()))
		return newPtr

	case reflect.Interface:
		if rv.IsNil() {
			return reflect.Zero(rv.Type())
		}
		copied := deepCopyValue(rv.Elem())
		result := reflect.New(rv.Type()).Elem()
		result.Set(copied)
		return result

	case reflect.Struct:
		newStruct := reflect.New(rv.Type()).Elem()
		for i := 0; i < rv.NumField(); i++ {
			if newStruct.Field(i).CanSet() {
				newStruct.Field(i).Set(deepCopyValue(rv.Field(i)))
			}
		}
		return newStruct

	case reflect.Slice:
		if rv.IsNil() {
			return reflect.Zero(rv.Type())
		}
		newSlice := reflect.MakeSlice(rv.Type(), rv.Len(), rv.Cap())
		for i := 0; i < rv.Len(); i++ {
			newSlice.Index(i).Set(deepCopyValue(rv.Index(i)))
		}
		return newSlice

	case reflect.Array:
		newArray := reflect.New(rv.Type()).Elem()
		for i := 0; i < rv.Len(); i++ {
			newArray.Index(i).Set(deepCopyValue(rv.Index(i)))
		}
		return newArray

	case reflect.Map:
		if rv.IsNil() {
			return reflect.Zero(rv.Type())
		}
		newMap := reflect.MakeMapWithSize(rv.Type(), rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			newKey := deepCopyValue(iter.Key())
			newVal := deepCopyValue(iter.Value())
			newMap.SetMapIndex(newKey, newVal)
		}
		return newMap

	case reflect.Chan:
		if rv.IsNil() {
			return reflect.Zero(rv.Type())
		}
		return reflect.MakeChan(rv.Type(), rv.Cap())

	case reflect.Func:
		return rv // Functions can't be deep copied

	default:
		newVal := reflect.New(rv.Type()).Elem()
		newVal.Set(rv)
		return newVal
	}
}

// ============================================================================
// Walking / Visiting
// ============================================================================

// WalkFunc is called for each value during traversal.
// path is the dot-separated path to the current value.
// Return false to skip children, true to continue.
type WalkFunc func(path string, v Value) bool

// Walk traverses a value depth-first and calls fn for each element.
func Walk(v any, fn WalkFunc) {
	if v == nil {
		return
	}
	walkValue("", reflect.ValueOf(v), fn)
}

func walkValue(path string, rv reflect.Value, fn WalkFunc) {
	if !rv.IsValid() {
		return
	}

	// Dereference pointers and interfaces
	for rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}

	// Call the walk function
	if !fn(path, Value{rv: rv}) {
		return // Skip children
	}

	// Recurse
	switch rv.Kind() {
	case reflect.Struct:
		for i := 0; i < rv.NumField(); i++ {
			field := rv.Type().Field(i)
			childPath := field.Name
			if path != "" {
				childPath = path + "." + field.Name
			}
			walkValue(childPath, rv.Field(i), fn)
		}

	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			childPath := fmt.Sprintf("%s[%d]", path, i)
			if path == "" {
				childPath = fmt.Sprintf("[%d]", i)
			}
			walkValue(childPath, rv.Index(i), fn)
		}

	case reflect.Map:
		iter := rv.MapRange()
		for iter.Next() {
			keyStr := fmt.Sprintf("%v", iter.Key().Interface())
			childPath := fmt.Sprintf("%s[%s]", path, keyStr)
			if path == "" {
				childPath = fmt.Sprintf("[%s]", keyStr)
			}
			walkValue(childPath, iter.Value(), fn)
		}
	}
}

// Visitor provides type-specific callbacks for traversal.
type Visitor struct {
	OnStruct func(path string, s Struct) bool
	OnSlice  func(path string, s Slice) bool
	OnMap    func(path string, m Map) bool
	OnValue  func(path string, v Value) bool
}

// Visit traverses a value using the Visitor callbacks.
func Visit(v any, visitor Visitor) {
	if v == nil {
		return
	}
	visitValue("", reflect.ValueOf(v), visitor)
}

func visitValue(path string, rv reflect.Value, visitor Visitor) {
	if !rv.IsValid() {
		return
	}

	// Dereference
	for rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}

	continueWalk := true

	switch rv.Kind() {
	case reflect.Struct:
		if visitor.OnStruct != nil {
			continueWalk = visitor.OnStruct(path, Struct{rv: rv, rt: rv.Type()})
		}
		if continueWalk {
			for i := 0; i < rv.NumField(); i++ {
				field := rv.Type().Field(i)
				childPath := field.Name
				if path != "" {
					childPath = path + "." + field.Name
				}
				visitValue(childPath, rv.Field(i), visitor)
			}
		}

	case reflect.Slice, reflect.Array:
		if visitor.OnSlice != nil {
			continueWalk = visitor.OnSlice(path, Slice{rv: rv, elemType: rv.Type().Elem()})
		}
		if continueWalk {
			for i := 0; i < rv.Len(); i++ {
				childPath := fmt.Sprintf("%s[%d]", path, i)
				if path == "" {
					childPath = fmt.Sprintf("[%d]", i)
				}
				visitValue(childPath, rv.Index(i), visitor)
			}
		}

	case reflect.Map:
		if visitor.OnMap != nil {
			continueWalk = visitor.OnMap(path, Map{rv: rv, keyType: rv.Type().Key(), valType: rv.Type().Elem()})
		}
		if continueWalk {
			iter := rv.MapRange()
			for iter.Next() {
				keyStr := fmt.Sprintf("%v", iter.Key().Interface())
				childPath := fmt.Sprintf("%s[%s]", path, keyStr)
				if path == "" {
					childPath = fmt.Sprintf("[%s]", keyStr)
				}
				visitValue(childPath, iter.Value(), visitor)
			}
		}

	default:
		if visitor.OnValue != nil {
			visitor.OnValue(path, Value{rv: rv})
		}
	}
}

// ============================================================================
// Struct ↔ Map Conversion
// ============================================================================

// StructToMap converts a struct to map[string]any.
// tagName specifies which tag to use for keys (e.g., "json").
// If tagName is empty, field names are used as keys.
func StructToMap(v any, tagName string) map[string]any {
	s := From(v).Struct()
	if tagName == "" {
		return s.ToMap()
	}
	return s.ToMapTag(tagName)
}

// MapToStruct populates a struct from a map.
// dst must be a pointer to a struct.
// tagName specifies which tag to use for matching keys.
func MapToStruct(m map[string]any, dst any, tagName string) error {
	dstVal := reflect.ValueOf(dst)
	if dstVal.Kind() != reflect.Pointer || dstVal.IsNil() {
		return fmt.Errorf("%w: dst must be non-nil pointer", ErrTypeMismatch)
	}

	dstVal = dstVal.Elem()
	if dstVal.Kind() != reflect.Struct {
		return fmt.Errorf("%w: dst must be pointer to struct", ErrTypeMismatch)
	}

	dstType := dstVal.Type()

	for key, value := range m {
		// Find matching field
		var fieldIndex int = -1

		for i := 0; i < dstType.NumField(); i++ {
			sf := dstType.Field(i)

			// Check tag first
			if tagName != "" {
				if tagVal := sf.Tag.Get(tagName); tagVal != "" {
					tagKey := tagVal
					for j, c := range tagVal {
						if c == ',' {
							tagKey = tagVal[:j]
							break
						}
					}
					if tagKey == key {
						fieldIndex = i
						break
					}
				}
			}

			// Check field name
			if sf.Name == key {
				fieldIndex = i
				break
			}
		}

		if fieldIndex < 0 {
			continue // No matching field
		}

		field := dstVal.Field(fieldIndex)
		if !field.CanSet() {
			continue
		}

		val := reflect.ValueOf(value)
		if !val.IsValid() {
			continue
		}

		if val.Type().AssignableTo(field.Type()) {
			field.Set(val)
		} else if val.Type().ConvertibleTo(field.Type()) {
			field.Set(val.Convert(field.Type()))
		}
	}

	return nil
}
