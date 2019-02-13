package deepmind

import (
	"github.com/pkg/errors"
	. "gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

// Layer is a set of neurons and corresponding activation
type Layer interface {
	Forward(x *Node, states States) (rv *Node, err error)
	Learnables() Nodes

	// If vs is nil, the initializer indicated in the Options will be used.
	Init(g *ExprGraph, dt tensor.Dtype, vs map[string][]float64) error

	// Get the name of the layer
	Name() string
	// Get the options of the layer
	Options() interface{}
}

type States map[string]*Node

func (s States) Get(name string) *Node {
	return s[name]
}

func (s States) Update(n *Node) {
	s[n.Name()] = n
}

func (s States) Len() int {
	return len(s)
}

//fully connected layer that has the operation: activate(x*w + b)
type FC struct {
	w *Node //with shap of [input, output]
	b *Node //with shap of [1, output]

	act   Activation
	initW InitWFn
	name  string
	opts  FCOpts
}

type FCOpts struct {
	InputSize  int
	OutputSize int
	// Sigmoid for example, see "active.go" for more activations.
	// Activation is optional,  default is Linear.
	Activation string

	//Gaussian(0.0, 0.08), see  "initializer.go" for more initializers.
	// Initializer is optional,  default is Uniform(-1,1).
	Initializer string

	// Probability of Dropout, it uses randomly zeroes out a *Tensor with a probability
	// drawn from a uniform distribution. Only float32 or float64 type supported.
	// Optional, default is zero, means
	Dropout float64
}

func NewFC(name string, opts FCOpts) (Layer, error) {
	if opts.Activation == "" {
		opts.Activation = "Linear"
	}
	act := Activations.Get(opts.Activation)
	if act == nil {
		return nil, errors.New("unknown activation name:" + opts.Activation)
	}

	initW, err := DefaultGetInitWFn(opts.Initializer, "Uniform(-1,1)")
	if err != nil {
		return nil, errors.Wrap(err, "GetInitWFn")
	}

	return &FC{
		act:   act,
		initW: initW,
		name:  name,
		opts:  opts,
	}, nil
}

func (l *FC) Name() string {
	return l.name
}

// If vs is nil, the initializer indicated in the Options will be used.
func (l *FC) Init(g *ExprGraph, dt tensor.Dtype, vs map[string][]float64) error {
	//w has shap of [input, output]
	//b has shap of [1, output]
	wShape := tensor.Shape{l.opts.InputSize, l.opts.OutputSize}
	bShape := tensor.Shape{l.opts.OutputSize}
	if l.opts.OutputSize < 2 {
		bShape = nil
	}

	if vs == nil {
		l.w = NewMatrix(g, dt, WithShape(l.opts.InputSize, l.opts.OutputSize), WithInit(l.initW), WithName(l.name+"_w"))
		if bShape == nil {
			l.b = NewScalar(g, dt, WithValue(F64ToAny(0.0, dt)), WithName(l.name+"_b"))
		} else {
			l.b = NewVector(g, dt, WithShape(bShape...), WithInit(Zeroes()), WithName(l.name+"_b"))
		}
		return nil
	}

	var err error
	if l.w, err = NodeFromMap(g, vs, dt, wShape, l.name+"_w"); err != nil {
		return errors.Wrap(err, "NodeFromMap")
	}
	if l.b, err = NodeFromMap(g, vs, dt, bShape, l.name+"_b"); err != nil {
		return errors.Wrap(err, "NodeFromMap")
	}
	return nil
}

func (l *FC) Options() interface{} {
	return l.opts
}

// activate(x*w + b)
// if dims of x > 2, x will be reshaped.
func (l *FC) Forward(x *Node, states States) (rv *Node, err error) {
	//dropout
	var dx *Node
	if l.opts.Dropout != 0.0 {
		dx, err = Dropout(x, l.opts.Dropout)
		if err != nil {
			return nil, errors.Wrap(err, "Dropout")
		}
		WithName(x.Name() + "_dropt")(dx)
	} else {
		dx = x
	}

	if rv, err = Fxwb(l.act, dx, l.w, l.b); err == nil {
		WithName(l.name + "_output")(rv)
	}
	return rv, err
}

// Learnables must be called after Init.
func (l *FC) Learnables() Nodes {
	return Nodes{l.w, l.b}
}

// Basic Recurrent Neural Network
type RNN struct {
	//h *Node // hidden state, with shape of [batch, hidden]
	w *Node // with shap of [input+hidden, hidden]
	b *Node //  with shap of [1, hidden]

	act   Activation
	initW InitWFn
	name  string
	dt    tensor.Dtype
	opts  RNNOpts
}

type RNNOpts struct {
	InputSize  int
	HiddenSize int
	// Sigmoid for example, see "active.go" for more activations.
	// Activation is optional,  default is Tanh.
	Activation string

	//Gaussian(0.0, 0.08), see  "initializer.go" for more initializers.
	// Initializer is optional,  default is Uniform(-1,1).
	Initializer string

	// Probability of Dropout, it uses randomly zeroes out a *Tensor with a probability
	// drawn from a uniform distribution. Only float32 or float64 type supported.
	// Optional, default is zero, means
	Dropout float64
}

func NewRNN(name string, opts RNNOpts) (Layer, error) {
	if opts.Activation == "" {
		opts.Activation = "Tanh"
	}
	act := Activations.Get(opts.Activation)
	if act == nil {
		return nil, errors.New("unknown activation name:" + opts.Activation)
	}

	initW, err := DefaultGetInitWFn(opts.Initializer, "Uniform(-1,1)")
	if err != nil {
		return nil, errors.Wrap(err, "GetInitWFn")
	}

	return &RNN{
		act:   act,
		initW: initW,
		name:  name,
		opts:  opts,
	}, nil
}

func (l *RNN) Name() string {
	return l.name
}

func (l *RNN) Options() interface{} {
	return l.opts
}

// If vs is nil, the initializer indicated in the Options will be used.
func (l *RNN) Init(g *ExprGraph, dt tensor.Dtype, vs map[string][]float64) error {
	l.dt = dt
	// w has shap of [input+hidden, hidden]
	// b has shap of [hidden]
	wShape := tensor.Shape{l.opts.InputSize + l.opts.HiddenSize, l.opts.HiddenSize}
	bShape := tensor.Shape{l.opts.HiddenSize}
	if l.opts.HiddenSize < 2 {
		bShape = nil
	}

	if vs == nil {
		l.w = NewMatrix(g, dt, WithShape(wShape...), WithInit(l.initW), WithName(l.name+"_w"))
		if bShape == nil {
			l.b = NewScalar(g, dt, WithValue(F64ToAny(0.0, dt)), WithName(l.name+"_b"))
		} else {
			l.b = NewVector(g, dt, WithShape(bShape...), WithInit(Zeroes()), WithName(l.name+"_b"))
		}
		return nil
	}

	var err error
	if l.w, err = NodeFromMap(g, vs, dt, wShape, l.name+"_w"); err != nil {
		return errors.Wrap(err, "NodeFromMap")
	}
	if l.b, err = NodeFromMap(g, vs, dt, bShape, l.name+"_b"); err != nil {
		return errors.Wrap(err, "NodeFromMap")
	}
	return nil
}

// h(dt+1) = activate([x, h(dt)]*w + b)
// if dims of x > 2, x will be reshaped.
func (l *RNN) Forward(x *Node, states States) (rv *Node, err error) {
	//dropout and auto reshape
	var dx *Node
	if l.opts.Dropout != 0.0 {
		dx, err = Dropout(x, l.opts.Dropout)
		if err != nil {
			return nil, errors.Wrap(err, "Dropout")
		}
		WithName(x.Name() + "_dropt")(dx)
	} else {
		dx = x
	}
	if dx, err = ReshapeToMatrix(dx); err != nil {
		return nil, errors.Wrap(err, "ReshapeToMatrix")
	}

	// init hidden state at the first call
	// h has shap of [batch, hidden]
	var h *Node
	if h = states.Get(l.name + "_h"); h == nil {
		h = NewMatrix(dx.Graph(), l.dt, WithShape(dx.Shape()[0], l.opts.HiddenSize), WithInit(Zeroes()), WithName(l.name+"_h"))
	}

	//concat hidden and input weights
	var xh *Node
	if xh, err = Concat(1, dx, h); err != nil {
		return nil, errors.Wrap(err, "Concat")
	}

	if rv, err = Fxwb(l.act, xh, l.w, l.b); err == nil {
		WithName(l.name + "_h")(rv)
		//update hidden parameter
		states.Update(rv)
	}

	return rv, err
}

// Learnables must be called after Init.
func (l *RNN) Learnables() Nodes {
	return Nodes{l.w, l.b}
}

// Long Short Term Memory
type LSTM struct {
	//h *Node //hidden state, with shap of [batch, hidden]
	//c *Node // cell state, with shap of [batch, hidden]

	//forget gate
	wf *Node //forget weights, with shap of [input + hidden, hidden]
	bf *Node //forget bias, with shap of [1, hidden]

	//input gate
	wi *Node //input weights, with shap of [input + hidden, hidden]
	bi *Node //input bias, with shap of [1, hidden]

	//output gate
	wo *Node //output weights, with shap of [input + hidden, hidden]
	bo *Node //output bias, with shap of [1, hidden]

	//cell
	wc *Node //cell weights, with shap of [input + hidden, hidden]
	bc *Node //cell bias, with shap of [1, hidden]

	initWf InitWFn
	initWi InitWFn
	initWo InitWFn
	initWc InitWFn

	act     Activation //tanh as usual
	sigmoid Activation // for forget, input and output gate
	name    string
	dt      tensor.Dtype
	opts    LSTMOpts
}

type LSTMOpts struct {
	InputSize  int
	HiddenSize int
	// Sigmoid for example, see "active.go" for more activations.
	// Activation is optional,  default is Tanh.
	Activation string

	//Gaussian(0.0, 0.08), see  "initializer.go" for more initializers.
	// Initializer is optional,  default is Uniform(-1,1).
	InitWf string
	InitWi string
	InitWo string
	InitWc string

	// Probability of Dropout, it uses randomly zeroes out a *Tensor with a probability
	// drawn from a uniform distribution. Only float32 or float64 type supported.
	// Optional, default is zero, means
	Dropout float64
}

func NewLSTM(name string, opts LSTMOpts) (Layer, error) {
	if opts.Activation == "" {
		opts.Activation = "Tanh"
	}
	act := Activations.Get(opts.Activation)
	if act == nil {
		return nil, errors.New("unknown activation name:" + opts.Activation)
	}

	initWf, err := DefaultGetInitWFn(opts.InitWf, "Uniform(-1,1)")
	if err != nil {
		return nil, errors.Wrap(err, "Get InitWf")
	}

	initWi, err := DefaultGetInitWFn(opts.InitWi, "Uniform(-1,1)")
	if err != nil {
		return nil, errors.Wrap(err, "Get InitWi")
	}

	initWo, err := DefaultGetInitWFn(opts.InitWo, "Uniform(-1,1)")
	if err != nil {
		return nil, errors.Wrap(err, "Get InitWo")
	}

	initWc, err := DefaultGetInitWFn(opts.InitWc, "Uniform(-1,1)")
	if err != nil {
		return nil, errors.Wrap(err, "Get InitWc")
	}

	return &LSTM{
		act:     act,
		sigmoid: Activations.Get("Sigmoid"),
		initWf:  initWf,
		initWi:  initWi,
		initWo:  initWo,
		initWc:  initWc,
		name:    name,
		opts:    opts,
	}, nil
}

// If vs is nil, the initializer indicated in the Options will be used,
func (l *LSTM) Init(g *ExprGraph, dt tensor.Dtype, vs map[string][]float64) error {
	l.dt = dt
	// w has shap of [input+hidden, hidden]
	// b has shap of [hidden]
	wShape := tensor.Shape{l.opts.InputSize + l.opts.HiddenSize, l.opts.HiddenSize}
	bShape := tensor.Shape{l.opts.HiddenSize}
	if l.opts.HiddenSize < 2 {
		bShape = nil
	}

	if vs == nil {
		l.wf = NewMatrix(g, dt, WithShape(wShape...), WithInit(l.initWf), WithName(l.name+"_wf"))
		l.wi = NewMatrix(g, dt, WithShape(wShape...), WithInit(l.initWi), WithName(l.name+"_wi"))
		l.wo = NewMatrix(g, dt, WithShape(wShape...), WithInit(l.initWo), WithName(l.name+"_wo"))
		l.wc = NewMatrix(g, dt, WithShape(wShape...), WithInit(l.initWc), WithName(l.name+"_wc"))

		if bShape == nil {
			l.bf = NewScalar(g, dt, WithValue(F64ToAny(0.0, dt)), WithName(l.name+"_bf"))
			l.bi = NewScalar(g, dt, WithValue(F64ToAny(0.0, dt)), WithName(l.name+"_bi"))
			l.bo = NewScalar(g, dt, WithValue(F64ToAny(0.0, dt)), WithName(l.name+"_bo"))
			l.bc = NewScalar(g, dt, WithValue(F64ToAny(0.0, dt)), WithName(l.name+"_bc"))
		} else {
			l.bf = NewVector(g, dt, WithShape(bShape...), WithInit(Zeroes()), WithName(l.name+"_bf"))
			l.bi = NewVector(g, dt, WithShape(bShape...), WithInit(Zeroes()), WithName(l.name+"_bi"))
			l.bo = NewVector(g, dt, WithShape(bShape...), WithInit(Zeroes()), WithName(l.name+"_bo"))
			l.bc = NewVector(g, dt, WithShape(bShape...), WithInit(Zeroes()), WithName(l.name+"_bc"))
		}

		return nil
	}

	//init from vs
	var err error
	if l.wf, err = NodeFromMap(g, vs, dt, wShape, l.name+"_wf"); err != nil {
		return errors.Wrap(err, "NodeFromMap")
	}
	if l.wi, err = NodeFromMap(g, vs, dt, wShape, l.name+"_wi"); err != nil {
		return errors.Wrap(err, "NodeFromMap")
	}
	if l.wo, err = NodeFromMap(g, vs, dt, wShape, l.name+"_wo"); err != nil {
		return errors.Wrap(err, "NodeFromMap")
	}
	if l.wc, err = NodeFromMap(g, vs, dt, wShape, l.name+"_wc"); err != nil {
		return errors.Wrap(err, "NodeFromMap")
	}
	//bias
	if l.bf, err = NodeFromMap(g, vs, dt, bShape, l.name+"_bf"); err != nil {
		return errors.Wrap(err, "NodeFromMap")
	}
	if l.bi, err = NodeFromMap(g, vs, dt, bShape, l.name+"_bi"); err != nil {
		return errors.Wrap(err, "NodeFromMap")
	}
	if l.bo, err = NodeFromMap(g, vs, dt, bShape, l.name+"_bo"); err != nil {
		return errors.Wrap(err, "NodeFromMap")
	}
	if l.bc, err = NodeFromMap(g, vs, dt, bShape, l.name+"_bc"); err != nil {
		return errors.Wrap(err, "NodeFromMap")
	}

	return nil
}

func (l *LSTM) Forward(x *Node, states States) (*Node, error) {
	// step 0, dropout and auto reshape
	var err error
	var dx *Node
	if l.opts.Dropout != 0.0 {
		dx, err = Dropout(x, l.opts.Dropout)
		if err != nil {
			return nil, errors.Wrap(err, "Dropout")
		}
		WithName(x.Name() + "_dropt")(dx)
	} else {
		dx = x
	}

	if dx, err = ReshapeToMatrix(dx); err != nil {
		return nil, errors.Wrap(err, "ReshapeToMatrix")
	}

	// step 1, concat dx and h
	// init hidden state and cell state at the first call
	// h and c have shap of [batch, hidden]
	var h, c, xh *Node
	if h = states.Get(l.name + "_h"); h == nil {
		h = NewMatrix(dx.Graph(), l.dt, WithShape(dx.Shape()[0], l.opts.HiddenSize), WithInit(Zeroes()), WithName(l.name+"_h"))
	}
	if c = states.Get(l.name + "_c"); c == nil {
		c = NewMatrix(dx.Graph(), l.dt, WithShape(dx.Shape()[0], l.opts.HiddenSize), WithInit(Zeroes()), WithName(l.name+"_c"))
	}

	if xh, err = Concat(1, dx, h); err != nil {
		return nil, errors.Wrap(err, "Concat")
	}

	// step 2, compute forgot parameter
	var forgot *Node
	if forgot, err = Fxwb(l.sigmoid, xh, l.wf, l.bf); err != nil {
		return nil, errors.Wrap(err, "forgot gate")
	}

	// step 3, compute input parameter
	var input *Node
	if input, err = Fxwb(l.sigmoid, xh, l.wi, l.bi); err != nil {
		return nil, errors.Wrap(err, "input gate")
	}

	// step 4, compute output parameter
	var output *Node
	if output, err = Fxwb(l.sigmoid, xh, l.wo, l.bo); err != nil {
		return nil, errors.Wrap(err, "output gate")
	}

	// step 5, update cell state
	// c_new = tanh(xh*wc + bc)
	// c = pointwise(forgot * c_old) + pointwise(input * c_new)
	var cf, ci, c_new *Node
	if c_new, err = Fxwb(l.act, xh, l.wc, l.bc); err != nil {
		return nil, errors.Wrap(err, "uptate cell")
	}
	if ci, err = HadamardProd(c_new, input); err != nil {
		return nil, errors.Wrap(err, "uptate cell")
	}
	if cf, err = HadamardProd(c, forgot); err != nil {
		return nil, errors.Wrap(err, "uptate cell")
	}
	if c, err = Add(cf, ci); err != nil {
		return nil, errors.Wrap(err, "uptate cell")
	}

	// step 6, update hidden state
	// h_new = pointwise(output * tanh(c_new) )
	if h, err = l.act.Activate(c); err != nil {
		return nil, errors.Wrap(err, l.act.Name())
	}
	if h, err = HadamardProd(h, output); err != nil {
		return nil, errors.Wrap(err, "uptate hidden state")
	}

	WithName(l.name + "_h")(h)
	WithName(l.name + "_c")(c)
	states.Update(h) //store hidden state
	states.Update(c) // store cell state

	return h, nil
}

func (l *LSTM) Learnables() Nodes {
	return Nodes{l.wf, l.wi, l.wo, l.wc, l.bf, l.bi, l.bo, l.bc}
}

func (l *LSTM) Name() string {
	return l.name
}

func (l *LSTM) Options() interface{} {
	return l.opts
}

// Gated Recurrent Unit
type GRU struct {
	//h  *Node //hidden state, with shap of [batch, hidden]
	wh *Node //hidden weights, with shap of [input + hidden, hidden]
	bh *Node //hidden bias, with shap of [1, hidden]

	//reset gate
	wr *Node //reset weights, with shap of [input + hidden, hidden]
	br *Node //reset bias, with shap of [1, hidden]

	//update gate
	wu *Node //update weights, with shap of [input + hidden, hidden]
	bu *Node //update bias, with shap of [1, hidden]

	initWh InitWFn
	initWr InitWFn
	initWu InitWFn

	act     Activation //tanh as usual
	sigmoid Activation // for forget, input and output gate
	name    string
	dt      tensor.Dtype
	opts    GRUOpts
}

type GRUOpts struct {
	InputSize  int
	HiddenSize int
	// Sigmoid for example, see "active.go" for more activations.
	// Activation is optional,  default is Tanh.
	Activation string

	//Gaussian(0.0, 0.08), see  "initializer.go" for more initializers.
	// Initializer is optional,  default is Uniform(-1,1).
	InitWh string
	InitWr string
	InitWu string

	// Probability of Dropout, it uses randomly zeroes out a *Tensor with a probability
	// drawn from a uniform distribution. Only float32 or float64 type supported.
	// Optional, default is zero, means
	Dropout float64
}

func NewGRU(name string, opts GRUOpts) (Layer, error) {
	if opts.Activation == "" {
		opts.Activation = "Tanh"
	}
	act := Activations.Get(opts.Activation)
	if act == nil {
		return nil, errors.New("unknown activation name:" + opts.Activation)
	}

	initWh, err := DefaultGetInitWFn(opts.InitWh, "Uniform(-1,1)")
	if err != nil {
		return nil, errors.Wrap(err, "Get InitWh")
	}

	initWr, err := DefaultGetInitWFn(opts.InitWr, "Uniform(-1,1)")
	if err != nil {
		return nil, errors.Wrap(err, "Get InitWr")
	}

	initWu, err := DefaultGetInitWFn(opts.InitWu, "Uniform(-1,1)")
	if err != nil {
		return nil, errors.Wrap(err, "Get InitWu")
	}

	return &GRU{
		act:     act,
		sigmoid: Activations.Get("Sigmoid"),
		initWh:  initWh,
		initWr:  initWr,
		initWu:  initWu,
		name:    name,
		opts:    opts,
	}, nil
}

// If vs is nil, the initializer indicated in the Options will be used,
func (l *GRU) Init(g *ExprGraph, dt tensor.Dtype, vs map[string][]float64) error {
	l.dt = dt
	// w has shap of [input+hidden, hidden]
	// b has shap of [hidden]
	wShape := tensor.Shape{l.opts.InputSize + l.opts.HiddenSize, l.opts.HiddenSize}
	bShape := tensor.Shape{l.opts.HiddenSize}
	if l.opts.HiddenSize < 2 {
		bShape = nil
	}

	if vs == nil {
		l.wh = NewMatrix(g, dt, WithShape(wShape...), WithInit(l.initWh), WithName(l.name+"_wh"))
		l.wr = NewMatrix(g, dt, WithShape(wShape...), WithInit(l.initWr), WithName(l.name+"_wr"))
		l.wu = NewMatrix(g, dt, WithShape(wShape...), WithInit(l.initWu), WithName(l.name+"_wu"))

		if bShape == nil {
			l.bh = NewScalar(g, dt, WithValue(F64ToAny(0.0, dt)), WithName(l.name+"_bh"))
			l.br = NewScalar(g, dt, WithValue(F64ToAny(0.0, dt)), WithName(l.name+"_br"))
			l.bu = NewScalar(g, dt, WithValue(F64ToAny(0.0, dt)), WithName(l.name+"_bu"))
		} else {
			l.bh = NewVector(g, dt, WithShape(bShape...), WithInit(Zeroes()), WithName(l.name+"_bh"))
			l.br = NewVector(g, dt, WithShape(bShape...), WithInit(Zeroes()), WithName(l.name+"_br"))
			l.bu = NewVector(g, dt, WithShape(bShape...), WithInit(Zeroes()), WithName(l.name+"_bu"))
		}

		return nil
	}

	//init from vs
	var err error
	if l.wh, err = NodeFromMap(g, vs, dt, wShape, l.name+"_wh"); err != nil {
		return errors.Wrap(err, "NodeFromMap")
	}
	if l.wr, err = NodeFromMap(g, vs, dt, wShape, l.name+"_wr"); err != nil {
		return errors.Wrap(err, "NodeFromMap")
	}
	if l.wu, err = NodeFromMap(g, vs, dt, wShape, l.name+"_wu"); err != nil {
		return errors.Wrap(err, "NodeFromMap")
	}

	//bias
	if l.bh, err = NodeFromMap(g, vs, dt, bShape, l.name+"_bh"); err != nil {
		return errors.Wrap(err, "NodeFromMap")
	}
	if l.br, err = NodeFromMap(g, vs, dt, bShape, l.name+"_br"); err != nil {
		return errors.Wrap(err, "NodeFromMap")
	}
	if l.bu, err = NodeFromMap(g, vs, dt, bShape, l.name+"_bu"); err != nil {
		return errors.Wrap(err, "NodeFromMap")
	}

	return nil
}

func (l *GRU) Forward(x *Node, states States) (*Node, error) {
	// step 0, dropout and auto reshape
	var err error
	var dx *Node
	if l.opts.Dropout != 0.0 {
		dx, err = Dropout(x, l.opts.Dropout)
		if err != nil {
			return nil, errors.Wrap(err, "Dropout")
		}
		WithName(x.Name() + "_dropt")(dx)
	} else {
		dx = x
	}
	if dx, err = ReshapeToMatrix(dx); err != nil {
		return nil, errors.Wrap(err, "ReshapeToMatrix")
	}

	//  step 1, concat dx and h
	// init hidden state and cell state at the first call
	// h and c have shap of [batch, hidden]
	var h, xh *Node
	if h = states.Get(l.name + "_h"); h == nil {
		h = NewMatrix(dx.Graph(), l.dt, WithShape(dx.Shape()[0], l.opts.HiddenSize), WithInit(Zeroes()), WithName(l.name+"_h"))
	}
	if xh, err = Concat(1, dx, h); err != nil {
		return nil, errors.Wrap(err, "Concat x and h")
	}

	// step 2, compute update parameter
	var update, forget *Node
	if update, err = Fxwb(l.sigmoid, xh, l.wu, l.bu); err != nil {
		return nil, errors.Wrap(err, "update gate")
	}
	if forget, err = OneSub(update); err != nil {
		return nil, errors.Wrap(err, "OneSub(update)")
	}

	// step 3, compute reset parameter
	var reset *Node
	if reset, err = Fxwb(l.sigmoid, xh, l.wr, l.br); err != nil {
		return nil, errors.Wrap(err, "reset gate")
	}

	// step 4, reset hidden state
	var hr *Node
	if hr, err = HadamardProd(h, reset); err != nil {
		return nil, errors.Wrap(err, "reset hidden state:")
	}

	// step 5, concat x and hr
	var xhr *Node
	if xhr, err = Concat(1, dx, hr); err != nil {
		return nil, errors.Wrap(err, "Concat x and hr")
	}

	// step 6, update hidden state
	// h_new = tanh(xhr*wh + bh)
	// h = pointwise((1 - update) * h_old) + pointwise(update * h_new)
	var hf, hi, h_new *Node
	if h_new, err = Fxwb(l.act, xhr, l.wh, l.bh); err != nil {
		return nil, errors.Wrap(err, "uptate hidden sate")
	}
	if hi, err = HadamardProd(h_new, update); err != nil {
		return nil, errors.Wrap(err, "uptate hidden sate")
	}
	if hf, err = HadamardProd(h, forget); err != nil {
		return nil, errors.Wrap(err, "uptate hidden sate")
	}
	if h, err = Add(hf, hi); err != nil {
		return nil, errors.Wrap(err, "uptate hidden sate")
	}

	WithName(l.name + "_h")(h)
	states.Update(h) //store hidden state

	return h, nil
}

func (l *GRU) Learnables() Nodes {
	return Nodes{l.wh, l.wr, l.wu, l.bh, l.br, l.bu}
}

func (l *GRU) Name() string {
	return l.name
}

func (l *GRU) Options() interface{} {
	return l.opts
}
