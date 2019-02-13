// Package inject provides utilities for mapping and injecting dependencies in various ways.
package inject

import (
	"fmt"
	"reflect"
)

type ErrType struct {
	reflect.Type
}

func (e ErrType) Error() string {
	return fmt.Sprintf("provided type not found: %v", e.Type)
}

type TypeGetter interface {
	// Returns the Value that is mapped to the current type. If Get returns a invalid Value,
	// an error should returned. (usually ErrNotFound)
	Get(reflect.Type) (reflect.Value, error)
}

// The GetterFunc type is an adapter to allow the use of
// ordinary functions asTypeGetter. If f is a function
// with the appropriate signature, GetterFunc(f) is a
// Handler that calls f.
type GetterFunc func(reflect.Type) (reflect.Value, error)

func (g GetterFunc) Get(t reflect.Type) (reflect.Value, error) {
	return g(t)
}

// TypeMapper represents an interface for mapping interface{} values based on type.
type TypeMapper interface {
	// Provides a possibility to directly insert a mapping based on type and value.
	// This makes it possible to directly map type arguments not possible to instantiate
	// with reflect like unidirectional channels.
	Set(reflect.Type, reflect.Value)
	TypeGetter
}

// Invoke attempts to call the interface{} provided as a function,
// providing dependencies for function arguments based on Type.
// Returns a slice of reflect.Value representing the returned values of the function.
// Returns an error if the injection fails.
// It panics if f is not a function.
func Invoke(f interface{}, g TypeGetter) ([]reflect.Value, error) {
	t := reflect.TypeOf(f)
	//Panic if t is not kind of Func
	var in = make([]reflect.Value, t.NumIn())
	for i := 0; i < t.NumIn(); i++ {
		argType := t.In(i)
		val, err := g.Get(argType)
		if err != nil {
			return nil, err
		}

		in[i] = val
	}

	return reflect.ValueOf(f).Call(in), nil
}

// mapping and injecting dependencies into function arguments.
type Injector struct {
	TypeMapper
}

// Invoke attempts to call the interface{} provided as a function,
// providing dependencies for function arguments based on Type.
// Returns a slice of reflect.Value representing the returned values of the function.
// Returns an error if the injection fails.
// It panics if f is not a function.
func (inj Injector) Invoke(f interface{}) ([]reflect.Value, error) {
	t := reflect.TypeOf(f)

	var in = make([]reflect.Value, t.NumIn()) //Panic if t is not kind of Func
	for i := 0; i < t.NumIn(); i++ {
		argType := t.In(i)
		val, err := inj.Get(argType)
		if err != nil {
			return nil, err
		}

		in[i] = val
	}

	return reflect.ValueOf(f).Call(in), nil
}

// Maps the concrete value of val to its dynamic type using reflect.TypeOf.
func (inj Injector) Map(val interface{}) {
	inj.Set(reflect.TypeOf(val), reflect.ValueOf(val))
}

// MapTo is useful when you mapping a interface.
// For example:
// type SpecialString interface {}
// var s SpecialString = "string"
// reflect.TypeOf(s) is string, not SpecialString
func (inj Injector) MapTo(val interface{}, ifacePtr interface{}) {
	inj.Set(InterfaceOf(ifacePtr), reflect.ValueOf(val))
}

// InterfaceOf dereferences a pointer to an Interface type.
// It panics if value is not an pointer to an interface.
func InterfaceOf(value interface{}) reflect.Type {
	t := reflect.TypeOf(value)

	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Interface {
		panic("Called inject.InterfaceOf with a value that is not a pointer to an interface. (*MyInterface)(nil)")
	}

	return t
}
