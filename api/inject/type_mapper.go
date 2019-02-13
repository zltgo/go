package inject

import (
	"reflect"
)

var (
	_ TypeMapper = TypeMap{}
	_ TypeMapper = new(TypePairs)
)

type TypeMap map[reflect.Type]reflect.Value

// Provides a possibility to directly insert a mapping based on type and value.
// This makes it possible to directly map type arguments not possible to instantiate
// with reflect like unidirectional channels.
func (m TypeMap) Set(typ reflect.Type, val reflect.Value) {
	m[typ] = val
}

// Returns the Value that is mapped to the current type. Returns a zeroed Value
// and an error if the Type has not been mapped.
// Map a nil interface{} will store a invalid value.
func (m TypeMap) Get(typ reflect.Type) (reflect.Value, error) {
	if v, ok := m[typ]; ok && v.IsValid() {
		return v, nil
	}

	// no concrete types found, try to find implementors
	// if typ is an interface
	//	if typ.Kind() == reflect.Interface {
	//		for k, v := range m {
	//			if k.Implements(typ) && v.IsValid() {
	//				return v, nil
	//			}
	//		}
	//	}

	return reflect.Value{}, &ErrType{typ}
}

/*******************TypePairs****************************/
type pair struct {
	typ reflect.Type
	val reflect.Value
}
type TypePairs []pair

func (m *TypePairs) Set(typ reflect.Type, val reflect.Value) {
	*m = append(*m, pair{typ, val})
}

func (m TypePairs) Get(typ reflect.Type) (reflect.Value, error) {
	// step 1, find if typ == pair.typ
	for _, pair := range m {
		if typ == pair.typ && pair.val.IsValid() {
			return pair.val, nil
		}
	}

	// step 2, try to find implementors
	// note that we can't combine step 1 and step 2.
	//	if typ.Kind() == reflect.Interface {
	//		for _, pair := range m {
	//			if pair.typ.Implements(typ) && pair.val.IsValid() {
	//				return pair.val, nil
	//			}
	//		}
	//	}

	return reflect.Value{}, &ErrType{typ}
}
