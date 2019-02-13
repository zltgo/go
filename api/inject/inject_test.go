package inject

import (
	"reflect"
	"testing"
)

type SpecialString interface {
}

type TestStruct struct {
	Dep1 string        `inject:"t" json:"-"`
	Dep2 SpecialString `inject`
	Dep3 string
}

type Greeter struct {
	Name string
}

func (g *Greeter) String() string {
	return "Hello, My name is" + g.Name
}

/* Test Helpers */
func expect(t *testing.T, a interface{}, b interface{}) {
	t.Helper()
	if a != b {
		t.Errorf("Expected %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

func refute(t *testing.T, a interface{}, b interface{}) {
	t.Helper()
	if a == b {
		t.Errorf("Did not expect %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

func Test_Invoke(t *testing.T) {

	inj := Injector{&TypePairs{}}
	dep := "some dependency"
	inj.Map(dep)
	dep2 := "another dep"
	inj.MapTo(dep2, (*SpecialString)(nil))
	dep3 := make(chan *SpecialString)
	dep4 := make(chan *SpecialString)
	typRecv := reflect.ChanOf(reflect.RecvDir, reflect.TypeOf(dep3).Elem())
	typSend := reflect.ChanOf(reflect.SendDir, reflect.TypeOf(dep4).Elem())

	inj.Set(typRecv, reflect.ValueOf(dep3))
	inj.Set(typSend, reflect.ValueOf(dep4))

	_, err := inj.Invoke(func(d1 string, d2 SpecialString, d3 <-chan *SpecialString, d4 chan<- *SpecialString) {
		expect(t, d1, dep)
		expect(t, d2, dep2)
		expect(t, reflect.TypeOf(d3).Elem(), reflect.TypeOf(dep3).Elem())
		expect(t, reflect.TypeOf(d4).Elem(), reflect.TypeOf(dep4).Elem())
		expect(t, reflect.TypeOf(d3).ChanDir(), reflect.RecvDir)
		expect(t, reflect.TypeOf(d4).ChanDir(), reflect.SendDir)
	})

	expect(t, err, nil)
}

func Test_InjectorInvokeReturnValues(t *testing.T) {
	inj := Injector{TypeMap{}}

	dep := "some dependency"
	inj.Map(dep)
	dep2 := "another dep"
	inj.MapTo(dep2, (*SpecialString)(nil))

	result, err := inj.Invoke(func(d1 string, d2 SpecialString) string {
		expect(t, d1, dep)
		expect(t, d2, dep2)
		return "Hello world"
	})

	expect(t, result[0].String(), "Hello world")
	expect(t, err, nil)
}

func Test_InterfaceOf(t *testing.T) {
	iType := InterfaceOf((*SpecialString)(nil))
	expect(t, iType.Kind(), reflect.Interface)

	iType = InterfaceOf((**SpecialString)(nil))
	expect(t, iType.Kind(), reflect.Interface)

	// Expecting nil
	defer func() {
		rec := recover()
		refute(t, rec, nil)
	}()
	iType = InterfaceOf((*testing.T)(nil))
}

func Test_InjectorSet(t *testing.T) {
	inj := make(TypeMap)
	typ := reflect.TypeOf("string")
	typSend := reflect.ChanOf(reflect.SendDir, typ)
	typRecv := reflect.ChanOf(reflect.RecvDir, typ)

	// instantiating unidirectional channels is not possible using reflect
	// http://golang.org/src/pkg/reflect/value.go?s=60463:60504#L2064
	chanRecv := reflect.MakeChan(reflect.ChanOf(reflect.BothDir, typ), 0)
	chanSend := reflect.MakeChan(reflect.ChanOf(reflect.BothDir, typ), 0)

	inj.Set(typSend, chanSend)
	inj.Set(typRecv, chanRecv)

	_, err := inj.Get(typSend)
	expect(t, err, nil)
	_, err = inj.Get(typRecv)
	expect(t, err, nil)
	_, err = inj.Get(chanSend.Type())

	expect(t, err.Error(), ErrType{chanSend.Type()}.Error())
}

func Test_InjectorGet(t *testing.T) {
	inj := Injector{new(TypePairs)}

	inj.Map("some dependency")

	_, err := inj.Get(reflect.TypeOf("string"))
	expect(t, err, nil)
	_, err = inj.Get(reflect.TypeOf(11))
	expect(t, err.Error(), ErrType{reflect.TypeOf(11)}.Error())
}

func Test_GetterFunc(t *testing.T) {
	f := func(t reflect.Type) (reflect.Value, error) {
		switch t {
		case reflect.TypeOf("string"):
			return reflect.ValueOf("some dependency"), nil
		case InterfaceOf((*SpecialString)(nil)):
			return reflect.ValueOf("another dep"), nil
		default:
			return reflect.Value{}, ErrType{t}
		}
	}

	fg := GetterFunc(f)

	if v, err := fg.Get(reflect.TypeOf("string")); err != nil || v.String() != "some dependency" {
		t.Error("something wrong:", v, err)
	}

	if v, err := fg.Get(InterfaceOf((*SpecialString)(nil))); err != nil || v.String() != "another dep" {
		t.Error("something wrong:", v, err)
	}
}

//func TestInjectImplementors(t *testing.T) {
//	inj := Injector{new(TypePairs)}
//	g := &Greeter{"Jeremy"}
//	inj.Map(g)

//	_, err := inj.Get(InterfaceOf((*fmt.Stringer)(nil)))
//	expect(t, err, nil)
//}

func simpleFunc(d1 string, d2 SpecialString) string {
	return "f1"
}

//func Benchmark_InvokeFunctor(b *testing.B) {
//	ft := NewFunctor(simpleFunc)
//	inj := Injector{new(TypePairs)}

//	dep := "some dependency"
//	inj.Map(dep)
//	dep2 := "another dep"
//	inj.MapTo(dep2, (*SpecialString)(nil))

//	b.ReportAllocs()
//	b.ResetTimer()
//	for i := 0; i < b.N; i++ {
//		if _, err := ft.Call(inj); err != nil {
//			b.Error(err)
//			return
//		}
//	}
//}

func Benchmark_InvokeTypeMap_Simple(b *testing.B) {
	inj := Injector{TypeMap{}}
	dep := "some dependency"
	inj.Map(dep)
	dep2 := "another dep"
	inj.MapTo(dep2, (*SpecialString)(nil))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := inj.Invoke(simpleFunc); err != nil {
			b.Error(err)
			return
		}
	}
}

func Benchmark_InvokeTypePair__Simple(b *testing.B) {
	inj := Injector{new(TypePairs)}
	dep := "some dependency"
	inj.Map(dep)
	dep2 := "another dep"
	inj.MapTo(dep2, (*SpecialString)(nil))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := inj.Invoke(simpleFunc); err != nil {
			b.Error(err)
			return
		}
	}
}

func Benchmark_InvokeGetterFunc_Simple(b *testing.B) {
	dep := "some dependency"
	var dep2 SpecialString = "another dep"

	f := func(t reflect.Type) (reflect.Value, error) {
		switch t {
		case reflect.TypeOf(""):
			return reflect.ValueOf(dep), nil
		case InterfaceOf((*SpecialString)(nil)):
			return reflect.ValueOf(dep2), nil
		default:
			return reflect.Value{}, ErrType{t}
		}
	}

	fg := GetterFunc(f)

	if v, err := fg.Get(InterfaceOf((*SpecialString)(nil))); err != nil || v.String() != "another dep" {
		b.Error("something wrong:", v, err)
		return
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Invoke(simpleFunc, fg); err != nil {
			b.Error(err)
			return
		}
	}
}

func complexFunc(i int, j int64, f32 float32, f64 float64, d1 string, d2 SpecialString) string {
	return "f1"
}

func Benchmark_InvokeTypeMap(b *testing.B) {
	inj := Injector{TypeMap{}}
	dep := "some dependency"
	inj.Map(dep)
	dep2 := "another dep"
	inj.MapTo(dep2, (*SpecialString)(nil))
	inj.Map(1)
	inj.Map(int64(2))
	inj.Map(float32(3.1))
	inj.Map(float64(3.14))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := inj.Invoke(complexFunc); err != nil {
			b.Error(err)
			return
		}
	}
}

func Benchmark_InvokeTypePair(b *testing.B) {
	inj := Injector{new(TypePairs)}
	dep := "some dependency"
	inj.Map(dep)
	dep2 := "another dep"
	inj.MapTo(dep2, (*SpecialString)(nil))
	inj.Map(1)
	inj.Map(int64(2))
	inj.Map(float32(3.1))
	inj.Map(float64(3.14))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := inj.Invoke(complexFunc); err != nil {
			b.Error(err)
			return
		}
	}
}

func Benchmark_InvokeGetterFunc(b *testing.B) {
	tString := reflect.TypeOf("string")
	tSpecialString := InterfaceOf((*SpecialString)(nil))
	tINT := reflect.TypeOf(0)
	tINT64 := reflect.TypeOf(int64(0))
	tFloat32 := reflect.TypeOf(float32(0))
	tFloat64 := reflect.TypeOf(float64(0))

	vString := reflect.ValueOf("some dependency")
	vSpecialString := reflect.ValueOf("another dep")
	vINT := reflect.ValueOf(1)
	vINT64 := reflect.ValueOf(int64(2))
	vFloat32 := reflect.ValueOf(float32(3.1))
	vFloat64 := reflect.ValueOf(float64(3.14))

	f := func(t reflect.Type) (reflect.Value, error) {
		switch t {
		case tString:
			return vString, nil
		case tSpecialString:
			return vSpecialString, nil
		case tINT:
			return vINT, nil
		case tINT64:
			return vINT64, nil
		case tFloat32:
			return vFloat32, nil
		case tFloat64:
			return vFloat64, nil
		default:
			return reflect.Value{}, ErrType{t}
		}
	}

	fg := GetterFunc(f)

	if v, err := fg.Get(InterfaceOf((*SpecialString)(nil))); err != nil || v.String() != "another dep" {
		b.Error("something wrong:", v, err)
		return
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Invoke(complexFunc, fg); err != nil {
			b.Error(err)
			return
		}
	}
}
