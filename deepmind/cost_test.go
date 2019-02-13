package deepmind

import (
	"math"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

func TestCost(t *testing.T) {
	g := NewGraph()

	Convey("should compute MeanSquared correctly", t, func() {
		x := NewTensor(g, tensor.Float32, 3, WithShape(2, 2, 2), WithBacking([]float64{0, 0, 1, 1, 0, 0, 1, 1}), WithName("x"))
		y := NewTensor(g, tensor.Float32, 3, WithShape(2, 2, 2), WithBacking([]float64{1, 1, 0, 0, 0, 0, 1, 4}), WithName("y"))
		cost, err := MeanSquared(x, y)
		So(err, ShouldBeNil)
		m := NewLispMachine(g, ExecuteFwdOnly())
		err = m.RunAll()
		So(err, ShouldBeNil)

		So(cost.Value().Data().(float32), ShouldEqual, 1.625)
	})

	Convey("should compute CrossEntropy correctly", t, func() {
		x := NewMatrix(g, tensor.Float64, WithShape(2, 2), WithBacking([]float64{0.1, 0.9, 0.2, 0.8}), WithName("x"))
		y := NewMatrix(g, tensor.Float64, WithShape(2, 2), WithBacking([]float64{0, 1, 0, 1}), WithName("y"))
		cost, err := CrossEntropy(x, y)
		So(err, ShouldBeNil)
		m := NewLispMachine(g, ExecuteFwdOnly())
		err = m.RunAll()
		So(err, ShouldBeNil)

		expect := (math.Log(0.9) + math.Log(0.8)) / (-4.0)
		So(cost.Value().Data().(float64), ShouldEqual, expect)
	})

	Convey("should compute BinaryCrossEntropy correctly", t, func() {
		output := NewMatrix(g, tensor.Float64, WithShape(2, 2), WithBacking([]float64{0.1, 0.2, 0.3, 0.4}), WithName("output"))
		target := NewMatrix(g, tensor.Float64, WithShape(2, 2), WithBacking([]float64{0.5, 0.6, 0.7, 0.8}), WithName("target"))
		cost, err := BinaryCrossEntropy(output, target)
		So(err, ShouldBeNil)
		m := NewLispMachine(g, ExecuteFwdOnly())
		err = m.RunAll()
		So(err, ShouldBeNil)

		expect := 0.5*math.Log(0.1) + 0.5*math.Log(1-0.1) + 0.6*math.Log(0.2) + 0.4*math.Log(1-0.2)
		expect += 0.7*math.Log(0.3) + 0.3*math.Log(1-0.3) + 0.8*math.Log(0.4) + 0.2*math.Log(1-0.4)
		expect = expect / (-4.0)
		So(cost.Value().Data().(float64), ShouldAlmostEqual, expect)
	})

	Convey("should compute losses correctly", t, func() {
		output1 := NewMatrix(g, tensor.Float64, WithShape(1, 2), WithBacking([]float64{0.1, 0.2}), WithName("output1"))
		output2 := NewMatrix(g, tensor.Float64, WithShape(1, 2), WithBacking([]float64{0.3, 0.4}), WithName("output2"))

		target1 := NewMatrix(g, tensor.Float64, WithShape(1, 2), WithBacking([]float64{0.5, 0.6}), WithName("target1"))
		target2 := NewMatrix(g, tensor.Float64, WithShape(1, 2), WithBacking([]float64{0.7, 0.8}), WithName("target2"))

		cost, err := Losses(Nodes{output1, output2}, Nodes{target1, target2}, BinaryCrossEntropy)
		So(err, ShouldBeNil)
		m := NewLispMachine(g, ExecuteFwdOnly())
		err = m.RunAll()
		So(err, ShouldBeNil)

		expect := 0.5*math.Log(0.1) + 0.5*math.Log(1-0.1) + 0.6*math.Log(0.2) + 0.4*math.Log(1-0.2)
		expect += 0.7*math.Log(0.3) + 0.3*math.Log(1-0.3) + 0.8*math.Log(0.4) + 0.2*math.Log(1-0.4)
		expect = expect / (-2.0)
		So(cost.Value().Data().(float64), ShouldEqual, expect)
	})

	Convey("should compute OneHotCE correctly", t, func() {
		output := NewVector(g, tensor.Float64, WithShape(4), WithBacking([]float64{0.1, 0.2, 0.3, 0.4}), WithName("output"))
		cost, err := OneHotCE(output, 3)
		So(err, ShouldBeNil)
		m := NewLispMachine(g, ExecuteFwdOnly())
		err = m.RunAll()
		So(err, ShouldBeNil)

		So(cost.Value().Data().(float64), ShouldAlmostEqual, -(math.Log(0.4)))
	})

	Convey("should compute OneHotCEBatch correctly", t, func() {
		output := NewMatrix(g, tensor.Float64, WithShape(2, 2), WithBacking([]float64{0.1, 0.9, 0.2, 0.8}), WithName("x"))
		cost, err := OneHotCEBatch(output, []int{1, 1})
		So(err, ShouldBeNil)
		m := NewLispMachine(g, ExecuteFwdOnly())
		err = m.RunAll()
		So(err, ShouldBeNil)

		expect := -(math.Log(0.9) + math.Log(0.8))
		So(cost.Value().Data().(float64), ShouldEqual, expect)
	})
}
