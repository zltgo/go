package deepmind

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/chewxy/hm"
	"github.com/pkg/errors"
	. "gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

var (
	OneF32     = NewConstant(float32(1.0))
	OneF64     = NewConstant(float64(1.0))
	OneInt     = NewConstant(int(1))
	OneInt64   = NewConstant(int64(1))
	OneInt32   = NewConstant(int32(1))
	shapeError = "cannot perform %v with shape %v and %v"
)

// CloneSelf Concat x on the provided axis n times.
func CloneSelf(axis int, x *Node, n int) (rv *Node, err error) {
	rv = x
	if n > 1 {
		clone := make(Nodes, n)
		for i := 0; i < n; i++ {
			clone[i] = x
		}
		//Concat batch size of b
		rv, err = Concat(axis, clone...)
	}
	return
}

// x supposed to have a batch size.
// AddBias adds b to evey sample of x.
func AddBias(x, b *Node) (rv *Node, err error) {
	//try to add first
	if rv, err = Add(x, b); err == nil {
		return rv, nil
	}

	xShape := x.Shape()
	bShape := b.Shape()
	bias := b

	switch {
	// x.Shape = {batch, n}, b.Shape = {1, n}
	case b.IsRowVec() && x.IsMatrix():
		if xShape[1] != bShape[1] {
			return nil, errors.Errorf(shapeError, "AddBias", xShape, bShape)
		}

		if bias, err = CloneSelf(0, b, xShape[0]); err != nil {
			return nil, errors.Wrap(err, "CloneSelf")
		}
	// x.Shape = {batch, n}, b.Shape = {batch, 1}
	case b.IsColVec() && x.IsMatrix():
		if xShape[0] != bShape[0] {
			return nil, errors.Errorf(shapeError, "AddBias", xShape, bShape)
		}

		if bias, err = CloneSelf(1, b, xShape[1]); err != nil {
			return nil, errors.Wrap(err, "CloneSelf")
		}
	//x.Shape = {batch_size, b.Shape()...}
	case x.Dims() == b.Dims()+1:
		// check the shape of x[0] and b
		for i := 0; i < b.Dims(); i++ {
			if xShape[i+1] != bShape[i] {
				return nil, errors.Errorf(shapeError, "AddBias", xShape, bShape)
			}
		}
		if bias, err = CloneSelf(0, b, xShape[0]); err != nil {
			return nil, errors.Wrap(err, "CloneSelf")
		}
	default:
		return nil, err
	}

	// Is b had been cloned, reshape it.
	if bias != b {
		if bias, err = Reshape(bias, xShape); err != nil {
			return nil, errors.Wrap(err, "Reshape")
		}
	}

	return Add(x, bias)
}

// if dims of x > 2, x will be reshaped to a matrix.
// ReshapeToMatrix is needed because trainning mode have batch size,  unlike product mode.
func ReshapeToMatrix(x *Node) (*Node, error) {
	switch x.Dims() {
	case 0:
		//unable to reshape a sacalar
		return x, nil
	case 1:
		return Reshape(x, tensor.Shape{1, x.Shape()[0]})
	case 2:
		return x, nil
	default:
		xs := x.Shape()
		col := 1
		for i := 1; i < xs.Dims(); i++ {
			col = col * xs[i]
		}
		return Reshape(x, tensor.Shape{xs[0], col})
	}
}

// perform f(xw+b)
func Fxwb(act Activation, x, w, b *Node) (*Node, error) {
	// auto reshape
	var err error
	if x, err = ReshapeToMatrix(x); err != nil {
		return nil, errors.Wrap(err, "ReshapeToMatrix")
	}

	var xw, xwb, fxwb *Node
	if xw, err = Mul(x, w); err != nil {
		return nil, errors.Wrap(err, "Mul")
	}

	if xwb, err = AddBias(xw, b); err != nil {
		return nil, errors.Wrap(err, "AddBias")
	}

	if fxwb, err = act.Activate(xwb); err != nil {
		return nil, errors.Wrap(err, act.Name())
	}
	return fxwb, nil
}

func dtypeOf(t hm.Type) (retVal tensor.Dtype, err error) {
	switch p := t.(type) {
	case tensor.Dtype:
		retVal = p
	case TensorType:
		return dtypeOf(p.Of)
	case hm.TypeVariable:
		err = errors.Errorf("instance %v does not have a dtype", p)
	default:
		err = errors.Errorf("dtypeOf not yet implemented for %v", p)
	}
	return
}

func DtypeOf(x *Node) (tensor.Dtype, error) {
	return dtypeOf(x.Type())
}

// perform 1-x
func OneSub(x *Node) (*Node, error) {
	dt, err := DtypeOf(x)
	if err != nil {
		return nil, err
	}

	var one *Node
	switch dt {
	case Float64:
		one = OneF64
	case Float32:
		one = OneF32
	case Int:
		one = OneInt
	case Int32:
		one = OneInt32
	case Int64:
		one = OneInt64
	default:
		return nil, errors.Errorf("OneSub not yet implemented for %v", dt)
	}

	return Sub(one, x)
}

func F64ToSlice(f64 []float64, dt tensor.Dtype) interface{} {
	switch dt {
	case tensor.Float64:
		return f64
	case tensor.Float32:
		rv := make([]float32, len(f64))
		for i, v := range f64 {
			rv[i] = float32(v)
		}
		return rv
	case tensor.Int:
		rv := make([]int, len(f64))
		for i, v := range f64 {
			rv[i] = int(v)
		}
		return rv
	case tensor.Int32:
		rv := make([]int32, len(f64))
		for i, v := range f64 {
			rv[i] = int32(v)
		}
		return rv
	case tensor.Int64:
		rv := make([]int64, len(f64))
		for i, v := range f64 {
			rv[i] = int64(v)
		}
		return rv
	case tensor.Bool:
		rv := make([]bool, len(f64))
		for i, v := range f64 {
			if v == 0.0 {
				rv[i] = false
			} else {
				rv[i] = true
			}
		}
		return rv
	default:
		panic(fmt.Sprintf("F64ToSlice not yet implemented for %v", dt))
	}
}

func GetBackingF64(n *Node) []float64 {
	val := n.Value().Data()
	v := reflect.ValueOf(val)

	if v.Kind() == reflect.Slice {
		numElems := v.Len()
		rv := make([]float64, numElems)
		for i := 0; i < numElems; i++ {
			rv[i] = AnyToF64(v.Index(i).Interface())
		}
		return rv
	}

	//scalar
	return []float64{AnyToF64(val)}
}

func AnyToF64(val interface{}) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case float32:
		f64 := float64(v)
		//  float64(v) != v, for example, float64(float32(0.1)) = 0.10000000149011612
		str := strconv.FormatFloat(f64, 'g', -1, 32)
		f64, _ = strconv.ParseFloat(str, 64)
		return f64
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case bool:
		if v {
			return 1.0
		}
		return 0.0
	default:
		panic(fmt.Sprintf("AnyToF64 not yet implemented for %T", v))
	}
}

func F64ToAny(v float64, dt tensor.Dtype) interface{} {
	switch dt {
	case tensor.Float64:
		return v
	case tensor.Float32:
		return float32(v)
	case tensor.Int:
		return int(v)
	case tensor.Int32:
		return int32(v)
	case tensor.Int64:
		return int64(v)
	case tensor.Bool:
		if v == 0.0 {
			return false
		}
		return true
	default:
		panic(fmt.Sprintf("F64ToAny not yet implemented for %v", dt))
	}
}

// length of backing can longer than node.TotalSize()
func WithBacking(f64 []float64) NodeConsOpt {
	return func(n *Node) {
		dt, err := DtypeOf(n)
		if err != nil {
			panic(err)
		}

		if n.IsScalar() {
			WithValue(F64ToAny(f64[0], dt))(n)
			return
		}

		back := F64ToSlice(f64, dt)
		v := tensor.New(tensor.WithShape(n.Shape()...), tensor.WithBacking(back))
		WithValue(v)(n)
	}
}

// if shape is nil, a scalar will be created.
func NodeFromMap(g *ExprGraph, vs map[string][]float64, dt tensor.Dtype, s tensor.Shape, name string) (*Node, error) {
	//find values in map by name
	back, ok := vs[name]
	if !ok {
		return nil, errors.Errorf("values of %s not found", name)
	}

	if s == nil {
		if len(back) > 1 {
			return nil, errors.Errorf("length of scalar expected to be 1,  got %v", len(back))
		}
		return NewScalar(g, dt, WithValue(F64ToAny(back[0], dt)), WithName(name)), nil
	}

	if len(back) != s.TotalSize() {
		return nil, errors.Errorf("shape mismatch, expected total size %v, got %v", s.TotalSize(), len(back))
	}

	return NewTensor(g, dt, s.Dims(), WithShape(s...), WithBacking(back), WithName(name)), nil
}
