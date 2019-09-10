package deepmind

import (
	"github.com/pkg/errors"
	. "gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

// combin a group of layers
type Model struct {
	Layers []Layer

	//init data from saver
	InitData map[string][]float64

	// weights and bias
	learnables  Nodes
	learnableMp map[string]*Node
}

func NewModel(layers ...Layer) *Model {
	// check layer names
	layerOpts := make(map[string]interface{})
	for _, l := range layers {
		if _, ok := layerOpts[l.Name()]; ok {
			panic("duplicated layer name: " + l.Name())
		}
		layerOpts[l.Name()] = l.Options()
	}

	return &Model{
		Layers:      layers,
		learnables:  Nodes{},
		learnableMp: make(map[string]*Node),
	}
}

// init weigths and bias at the beginning.
func (m *Model) Init(g *ExprGraph, dt tensor.Dtype) error {
	for _, l := range m.Layers {
		// if m.Data is nil, use the initialize function instead.
		if err := l.Init(g, dt, m.InitData); err != nil {
			return errors.Wrap(err, l.Name())
		}

		//init slice and map of learnables
		lbs := l.Learnables()
		m.learnables = append(m.learnables, lbs...)

		for _, n := range lbs {
			if _, ok := m.learnableMp[n.Name()]; ok {
				return errors.New("duplicated node name: " + n.Name())
			}
			m.learnableMp[n.Name()] = n
		}
	}
	return nil
}

// states must be empty in the beginning.
// states stores hidden state in the layers if necessary.
func (m *Model) Forward(x *Node, states States) (rv *Node, err error) {
	rv = x
	for _, l := range m.Layers {
		if rv, err = l.Forward(rv, states); err != nil {
			return nil, errors.Wrap(err, l.Name())
		}
	}
	return rv, nil
}

// len(ns) = number of steps
func (m *Model) StepForward(ns Nodes) (rv Nodes, err error) {
	states := States{}
	var tmp *Node
	for _, x := range ns {
		if tmp, err = m.Forward(x, states); err != nil {
			return nil, err
		}
		rv = append(rv, tmp)
	}
	return rv, nil
}

//get all learnable nodes.
func (m *Model) Learnables() Nodes {
	return m.learnables
}

func (m *Model) LearnablesGrad() []ValueGrad {
	rv := make([]ValueGrad, len(m.learnables))
	for i := 0; i < len(m.learnables); i++ {
		rv[i] = m.learnables[i]
	}
	return rv
}

// get learnable node by name.
func (m *Model) GetNode(name string) *Node {
	return m.learnableMp[name]
}
