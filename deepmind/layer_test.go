package deepmind

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

type forward interface {
	Forward(x *Node, states States) (rv *Node, err error)
}

func runForward(g *ExprGraph, f forward, x *Node) (rv *Node, err error) {
	states := States{}
	rv, err = f.Forward(x, states)
	if err != nil {
		return nil, err
	}

	m := NewLispMachine(g, ExecuteFwdOnly())
	if err = m.RunAll(); err != nil {
		return nil, err
	}
	return rv, nil
}

func TestFC(t *testing.T) {
	vs := map[string][]float64{
		"layer1_w": {0.1, 0.2, 0.3, 0.4, 0.5, 0.6},
		"layer1_b": {0.1, -0.1},
	}
	Convey("should perform Forward correctly", t, func() {
		fc, err := NewFC("layer1", FCOpts{
			InputSize:  3,
			OutputSize: 2,
			Activation: "Tanh",
		})
		So(err, ShouldBeNil)

		g := NewGraph()
		err = fc.Init(g, tensor.Float32, vs)
		So(err, ShouldBeNil)

		x := NewVector(g, tensor.Float32, WithShape(3), WithBacking([]float64{0, 0, 1}))
		rv, err := runForward(g, fc, x)

		So(err, ShouldBeNil)
		got := rv.Value().Data().([]float32)
		So(got, ShouldResemble, []float32{0.53704957, 0.46211717})
	})

	Convey("should perform Forward correctly with batch", t, func() {
		fc, err := NewFC("layer1", FCOpts{
			InputSize:  3,
			OutputSize: 2,
			Activation: "Tanh",
		})
		So(err, ShouldBeNil)

		g := NewGraph()
		err = fc.Init(g, tensor.Float32, vs)
		So(err, ShouldBeNil)

		x := NewMatrix(g, tensor.Float32, WithShape(3, 3), WithBacking([]float64{0, 0, 1, 0, 1, 0, 1, 0, 0}))
		rv, err := runForward(g, fc, x)

		So(err, ShouldBeNil)
		got := rv.Value().Data().([]float32)
		So(got, ShouldResemble, []float32{0.53704957, 0.46211717, 0.37994897, 0.29131263, 0.19737533, 0.099667996})
	})
}

func TestRNN(t *testing.T) {
	vs := map[string][]float64{
		"layer1_w": {0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8},
		"layer1_b": {0.1, -0.1},
	}
	Convey("should perform Forward correctly", t, func() {
		rnn, err := NewRNN("layer1", RNNOpts{
			InputSize:  2,
			HiddenSize: 2,
			Activation: "Tanh",
		})
		So(err, ShouldBeNil)

		g := NewGraph()
		err = rnn.Init(g, tensor.Float32, vs)
		So(err, ShouldBeNil)

		x := NewMatrix(g, tensor.Float32, WithShape(3, 2), WithBacking([]float64{1, 0, 0, 1, 1, 1}))
		rv, err := runForward(g, rnn, x)

		t.Log(rv.Value())
		So(err, ShouldBeNil)
		got := rv.Value().Data().([]float32)
		So(got, ShouldResemble, []float32{0.19737533, 0.099667996, 0.37994897, 0.29131263, 0.46211717, 0.46211717})
	})
}
