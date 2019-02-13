package deepmind

import (
	"io/ioutil"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

func TestSaver(t *testing.T) {
	os.RemoveAll("./testDir")
	os.MkdirAll("./testDir", os.ModePerm)

	vs := map[string][]float64{
		"layer1_w": {0.1, 0.2, 0.3, 0.4, 0.5, 0.6},
		"layer1_b": {0.1, -0.1},
		"layer2_w": {1.0, 2.0},
		"layer2_b": {0.1},
	}

	Convey("should save model correctly", t, func() {
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

		s := NewJsonSaver("./testDir")
		err = s.Save(m)
		So(err, ShouldBeNil)
		So(readFIle("./testDir/data.json"), ShouldEqual, `{"layer1_b":[0.1,-0.1],"layer1_w":[0.1,0.2,0.3,0.4,0.5,0.6],"layer2_b":[0.1],"layer2_w":[1,2]}
`)
		So(readFIle("./testDir/graph.json"), ShouldEqual, `{
    "Names": [
        "layer1",
        "layer2"
    ],
    "Opts": {
        "layer1": {
            "Activation": "Tanh",
            "Dropout": 0,
            "Initializer": "",
            "InputSize": 3,
            "OutputSize": 2,
            "_struct_name": "deepmind.FCOpts"
        },
        "layer2": {
            "Activation": "ReLU",
            "Dropout": 0,
            "Initializer": "",
            "InputSize": 2,
            "OutputSize": 1,
            "_struct_name": "deepmind.FCOpts"
        }
    },
    "_struct_name": "deepmind.LayerOpts"
}`)
	})

	Convey("should load model correctly", t, func() {
		s := NewJsonSaver("./testDir")
		m, err := s.Load()
		So(err, ShouldBeNil)
		So(m.InitData, ShouldResemble, vs)
		So(m.Layers, ShouldHaveLength, 2)
		So(m.Layers[0].Name(), ShouldEqual, "layer1")
		So(m.Layers[1].Name(), ShouldEqual, "layer2")
		So(m.Layers[0].Options(), ShouldResemble, FCOpts{
			InputSize:  3,
			OutputSize: 2,
			Activation: "Tanh",
		})
		So(m.Layers[1].Options(), ShouldResemble, FCOpts{
			InputSize:  2,
			OutputSize: 1,
			Activation: "ReLU",
		})
	})
}

func readFIle(path string) string {
	fp, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return err.Error()
	}
	defer fp.Close()

	b, err := ioutil.ReadAll(fp)
	if err != nil {
		return err.Error()
	}
	return string(b)
}
