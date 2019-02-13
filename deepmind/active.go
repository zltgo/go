package deepmind

import (
	. "gorgonia.org/gorgonia"
)

var Activations activationMap

func init() {
	Activations = make(map[string]Activation, 5)
	// Default is Linear
	Activations.Register(NewActivation("", Linear))
	Activations.Register(NewActivation("Sigmoid", Sigmoid))
	Activations.Register(NewActivation("Tanh", Tanh))
	Activations.Register(NewActivation("ReLU", Rectify))
	Activations.Register(NewActivation("Linear", Linear))
	Activations.Register(NewActivation("SoftMax", SoftMax))
}

type Activation interface {
	Activate(x *Node) (*Node, error)
	Name() string
}

// Wrap a activate function to Activation
type activation struct {
	name string
	act  func(x *Node) (*Node, error)
}

func (a activation) Name() string {
	return a.name
}

func (a activation) Activate(x *Node) (*Node, error) {
	return a.act(x)
}

// create a Activation by a activate function
func NewActivation(name string, fn func(x *Node) (*Node, error)) Activation {
	return activation{
		name: name,
		act:  fn,
	}
}

//Cache all the activations
type activationMap map[string]Activation

// Register replaces any existing activations.
// It is not concurrent safe, use in init of your package
func (m activationMap) Register(a Activation) {
	m[a.Name()] = a
}

// Get returns nil if name does not exist.
func (m activationMap) Get(name string) Activation {
	return m[name]
}

//there is no activation
func Linear(x *Node) (*Node, error) {
	return x, nil
}
