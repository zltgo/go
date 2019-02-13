package deepmind

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

func runFxwb(g *ExprGraph, act Activation, x, w, b *Node) (rv *Node, err error) {
	rv, err = Fxwb(act, x, w, b)
	if err != nil {
		return nil, err
	}

	m := NewLispMachine(g, ExecuteFwdOnly())
	if err = m.RunAll(); err != nil {
		return nil, err
	}
	return rv, nil
}

func TestFxwb(t *testing.T) {
	Convey("should perform Fxwb correctly", t, func() {
		g := NewGraph()
		tanh := Activations.Get("Tanh")
		x := NewVector(g, tensor.Float32, WithShape(3), WithBacking([]float64{0, 0, 1}))
		w := NewMatrix(g, tensor.Float32, WithShape(3, 2), WithBacking([]float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6}))
		b := NewVector(g, tensor.Float32, WithShape(2), WithBacking([]float64{0.1, -0.1}))
		rv, err := runFxwb(g, tanh, x, w, b)

		So(err, ShouldBeNil)
		got := rv.Value().Data().([]float32)
		So(got[0], ShouldEqual, 0.53704957)
		So(got[1], ShouldEqual, 0.46211716)
	})
}

func TestReshapeToMatrix(t *testing.T) {
	g := NewGraph()
	Convey("reshape scalar", t, func() {
		x := NewScalar(g, tensor.Int, WithValue(1))
		rx, err := ReshapeToMatrix(x)
		So(err, ShouldBeNil)
		So(rx, ShouldEqual, x)
	})

	Convey("reshape vector", t, func() {
		x := NewVector(g, tensor.Int, WithShape(3), WithBacking([]float64{0, 0, 0}))
		rx, err := ReshapeToMatrix(x)
		So(err, ShouldBeNil)
		So(rx.Shape(), ShouldResemble, tensor.Shape{1, 3})
	})

	Convey("reshape matrix", t, func() {
		x := NewMatrix(g, tensor.Int, WithShape(3, 2), WithInit(RangedFrom(0)))
		rx, err := ReshapeToMatrix(x)
		So(err, ShouldBeNil)
		So(rx, ShouldEqual, x)
	})

	Convey("reshape tensor", t, func() {
		x := NewTensor(g, tensor.Int, 5, WithShape(3, 2, 2, 1, 2), WithInit(RangedFrom(0)))
		rx, err := ReshapeToMatrix(x)
		So(err, ShouldBeNil)
		So(rx.Shape(), ShouldResemble, tensor.Shape{3, 8})
	})
}

func runAddbias(g *ExprGraph, x, b *Node) (rv *Node, err error) {
	rv, err = AddBias(x, b)
	if err != nil {
		return nil, err
	}

	m := NewLispMachine(g, ExecuteFwdOnly())
	if err = m.RunAll(); err != nil {
		return nil, err
	}
	return rv, nil
}

func TestAddBias(t *testing.T) {
	Convey("bias is a scalar", t, func() {
		g := NewGraph()
		x := NewMatrix(g, tensor.Float32, WithShape(3, 2), WithInit(RangedFrom(0)))
		b := NewScalar(g, tensor.Float32, WithValue(float32(1.0)))

		xb, err := runAddbias(g, x, b)
		So(err, ShouldBeNil)
		So(xb.Value().Data().([]float32), ShouldResemble, tensor.Range(tensor.Float32, 1, 7))
	})

	Convey("bias is a vector", t, func() {
		g := NewGraph()
		x := NewMatrix(g, tensor.Float32, WithShape(3, 2), WithInit(RangedFrom(0)))
		b := NewVector(g, tensor.Float32, WithShape(2), WithBacking([]float64{1.0, 2.0}))

		xb, err := runAddbias(g, x, b)
		So(err, ShouldBeNil)
		So(xb.Value().Data().([]float32), ShouldResemble, []float32{1, 3, 3, 5, 5, 7})

		// batch size = 1
		x = NewMatrix(g, tensor.Float32, WithShape(1, 2), WithInit(RangedFrom(0)))
		xb, err = runAddbias(g, x, b)
		So(err, ShouldBeNil)
		So(xb.Value().Data().([]float32), ShouldResemble, []float32{1, 3})

		//shape error
		x = NewMatrix(g, tensor.Float32, WithShape(1, 3), WithInit(RangedFrom(0)))
		b = NewVector(g, tensor.Float32, WithShape(2), WithBacking([]float64{1.0, 2.0}))
		_, err = runAddbias(g, x, b)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "cannot perform AddBias with shape (1, 3) and (2)")
	})

	Convey("bias is a IsColVec", t, func() {
		g := NewGraph()
		x := NewMatrix(g, tensor.Float32, WithShape(3, 2), WithInit(RangedFrom(0)))
		b := NewMatrix(g, tensor.Float32, WithShape(3, 1), WithBacking([]float64{1, 2, 3}))

		xb, err := runAddbias(g, x, b)
		So(err, ShouldBeNil)
		So(xb.Value().Data().([]float32), ShouldResemble, []float32{1, 2, 4, 5, 7, 8})

		//shape error
		b = NewMatrix(g, tensor.Float32, WithShape(2, 1), WithBacking([]float64{1, 2}))
		_, err = runAddbias(g, x, b)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "cannot perform AddBias with shape (3, 2) and (2, 1)")
	})

	Convey("bias is a IsRowVec", t, func() {
		g := NewGraph()
		x := NewMatrix(g, tensor.Float32, WithShape(3, 2), WithInit(RangedFrom(0)))
		b := NewMatrix(g, tensor.Float32, WithShape(1, 2), WithBacking([]float64{1, 2}))

		xb, err := runAddbias(g, x, b)
		So(err, ShouldBeNil)
		So(xb.Value().Data().([]float32), ShouldResemble, []float32{1, 3, 3, 5, 5, 7})

		//shape error
		b = NewMatrix(g, tensor.Float32, WithShape(1, 3), WithBacking([]float64{1, 2, 3}))
		_, err = runAddbias(g, x, b)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "cannot perform AddBias with shape (3, 2) and (1, 3)")
	})
}

func TestF64ToAny(t *testing.T) {
	Convey("float64 to implemented types", t, func() {
		So(F64ToAny(1, tensor.Float64), ShouldEqual, 1.0)
		So(F64ToAny(2, tensor.Float32), ShouldEqual, float32(2))
		So(F64ToAny(3, tensor.Int), ShouldEqual, 3)
		So(F64ToAny(4, tensor.Int32), ShouldEqual, int32(4))
		So(F64ToAny(5, tensor.Int64), ShouldEqual, int64(5))
		So(F64ToAny(6, tensor.Bool), ShouldEqual, true)
		So(F64ToAny(0, tensor.Bool), ShouldEqual, false)
	})
	Convey("should panic", t, func() {
		So(func() { F64ToAny(0, tensor.Int8) }, ShouldPanicWith, "F64ToAny not yet implemented for int8")
	})
}

func TestF64ToSlice(t *testing.T) {
	Convey("float64 to implemented types", t, func() {
		So(F64ToSlice([]float64{0, 1}, tensor.Float64), ShouldResemble, []float64{0, 1})
		So(F64ToSlice([]float64{0, 1}, tensor.Float32), ShouldResemble, []float32{0, 1})
		So(F64ToSlice([]float64{0, 1}, tensor.Int), ShouldResemble, []int{0, 1})
		So(F64ToSlice([]float64{0, 1}, tensor.Int32), ShouldResemble, []int32{0, 1})
		So(F64ToSlice([]float64{0, 1}, tensor.Int64), ShouldResemble, []int64{0, 1})
		So(F64ToSlice([]float64{0, 1}, tensor.Bool), ShouldResemble, []bool{false, true})
	})
	Convey("should panic", t, func() {
		So(func() { F64ToSlice([]float64{0, 1}, tensor.Int8) }, ShouldPanicWith, "F64ToSlice not yet implemented for int8")
	})
}

func TestWithBacking(t *testing.T) {
	g := NewGraph()
	Convey("float64 to implemented types", t, func() {
		n := NewTensor(g, tensor.Float32, 5, WithShape(1, 2, 2, 1, 2), WithBacking([]float64{1, 2, 3, 4, 5, 6, 7, 8}))
		So(n.Value().Data().([]float32), ShouldResemble, []float32{1, 2, 3, 4, 5, 6, 7, 8})

		n = NewTensor(g, tensor.Int, 5, WithShape(1, 2, 2, 1, 2), WithBacking([]float64{1, 2, 3, 4, 5, 6, 7, 8}))
		So(n.Value().Data().([]int), ShouldResemble, []int{1, 2, 3, 4, 5, 6, 7, 8})

		n = NewScalar(g, tensor.Int32, WithBacking([]float64{1}))
		So(n.Value().Data().(int32), ShouldEqual, 1)
	})
	Convey("should panic", t, func() {
		So(func() {
			NewTensor(g, tensor.Int8, 1, WithShape(2), WithBacking([]float64{1, 2}))
		}, ShouldPanicWith, "F64ToSlice not yet implemented for int8")
	})
}

func TestNodeFromMap(t *testing.T) {
	g := NewGraph()
	vs := make(map[string][]float64)
	vs["float32"] = []float64{1, 2, 3, 4, 5, 6, 7, 8}
	vs["int"] = []float64{2, 2, 3, 4, 5, 6}
	vs["scalar"] = []float64{3}

	Convey("should get nodes from map correctly", t, func() {
		n, err := NodeFromMap(g, vs, tensor.Float32, tensor.Shape{1, 2, 2, 1, 2}, "float32")
		So(err, ShouldBeNil)
		So(n.Value().Data().([]float32), ShouldResemble, []float32{1, 2, 3, 4, 5, 6, 7, 8})

		n, err = NodeFromMap(g, vs, tensor.Int, tensor.Shape{1, 6}, "int")
		So(err, ShouldBeNil)
		So(n.Value().Data().([]int), ShouldResemble, []int{2, 2, 3, 4, 5, 6})

		n, err = NodeFromMap(g, vs, tensor.Int, nil, "scalar")
		So(err, ShouldBeNil)
		So(n.Value().Data().(int), ShouldEqual, 3)
	})

	Convey("shape mismatch", t, func() {
		_, err := NodeFromMap(g, vs, tensor.Float32, tensor.Shape{1, 2, 2, 1, 3}, "float32")
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "shape mismatch, expected total size 12, got 8")

		_, err = NodeFromMap(g, vs, tensor.Int, nil, "int")
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "length of scalar expected to be 1,  got 6")
	})
}

func TestGetBackingF64(t *testing.T) {
	g := NewGraph()
	vs := make(map[string][]float64)
	vs["float64"] = []float64{0.01, 0.02, 0.03, 0.04, 0.05, 0.06, 0.07, 0.08}
	vs["float32"] = []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8}
	vs["int"] = []float64{1, 2, 3, 4, 5, 6}
	vs["bool"] = []float64{1, 0, 0, 1, 0}
	vs["scalar"] = []float64{3}

	Convey("should get backing from nodes correctly", t, func() {
		n, err := NodeFromMap(g, vs, tensor.Float64, tensor.Shape{1, 2, 2, 1, 2}, "float64")
		So(err, ShouldBeNil)
		So(GetBackingF64(n), ShouldResemble, vs["float64"])

		n, err = NodeFromMap(g, vs, tensor.Float32, tensor.Shape{1, 2, 2, 1, 2}, "float32")
		So(err, ShouldBeNil)
		So(GetBackingF64(n), ShouldResemble, vs["float32"])

		n, err = NodeFromMap(g, vs, tensor.Int, tensor.Shape{1, 6}, "int")
		So(err, ShouldBeNil)
		So(GetBackingF64(n), ShouldResemble, vs["int"])

		n, err = NodeFromMap(g, vs, tensor.Bool, tensor.Shape{1, 5}, "bool")
		So(err, ShouldBeNil)
		So(GetBackingF64(n), ShouldResemble, vs["bool"])

		n, err = NodeFromMap(g, vs, tensor.Int, nil, "scalar")
		So(err, ShouldBeNil)
		So(GetBackingF64(n), ShouldResemble, vs["scalar"])
	})

	Convey("should panic", t, func() {
		n := NewScalar(g, tensor.Byte, WithValue(byte(1)))
		So(func() { GetBackingF64(n) }, ShouldPanicWith, "AnyToF64 not yet implemented for uint8")
	})
}
