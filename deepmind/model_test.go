package deepmind

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

func TestModel(t *testing.T) {
	vs := map[string][]float64{
		"layer1_w": {0.1, 0.2, 0.3, 0.4, 0.5, 0.6},
		"layer1_b": {0.1, -0.1},
		"layer2_w": {1.0, 2.0},
		"layer2_b": {0.1},
	}

	Convey("should perform Forward correctly", t, func() {
		layer1, err := NewFC("layer1", FCOpts{
			InputSize:  3,
			OutputSize: 2,
			Activation: "Tanh",
		})
		So(err, ShouldBeNil)

		layer2, err := NewFC("layer2", FCOpts{
			InputSize:  2,
			OutputSize: 1,
			Activation: "ReLU",
		})
		So(err, ShouldBeNil)

		g := NewGraph()
		m := NewModel(layer1, layer2)
		m.InitData = vs
		m.Init(g, tensor.Float32)

		x := NewVector(g, tensor.Float32, WithShape(3), WithBacking([]float64{0, 0, 1}))
		rv, err := runForward(g, m, x)

		So(err, ShouldBeNil)
		got := rv.Value().Data().([]float32)
		So(got[0], ShouldEqual, 1.561284)
	})
}
