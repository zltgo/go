package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"strconv"
	"time"

	. "github.com/zltgo/deepmind"
	. "gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

var (
	// gradient update stuff
	l2reg      = 0.000001
	learnrate  = 0.001
	clipVal    = 5.0
	trainIter  = 5000
	steps      = 10 //time steps
	viewIter   = 100
	hiddenSize = 30
	batchSize  = 100
)

func main() {
	rand.Seed(time.Now().Unix())

	saver := NewJsonSaver("./save")
	model, err := saver.Load()
	if os.IsNotExist(err) {
		model, err = NewGRUModel()
	}
	handleError(err, "failed to load or create model: ")

	g := NewGraph()
	handleError(model.Init(g, tensor.Float64), "init model")

	x := make(Nodes, steps)
	y := make(Nodes, steps)
	for i := 0; i < steps; i++ {
		x[i] = NewMatrix(g, tensor.Float64, WithShape(batchSize, 2), WithName("x"+strconv.FormatInt(int64(i), 10)))
		y[i] = NewMatrix(g, tensor.Float64, WithShape(batchSize, 2), WithName("y"+strconv.FormatInt(int64(i), 10)))
	}

	output, err := model.StepForward(x)
	handleError(err, "model.Forward: ")

	cost, err := Losses(output[steps/2:steps], y[steps/2:steps], MeanSquared)
	//cost, err := MeanSquared(output[steps-1], y[steps-1])
	handleError(err, "MeanSquared: ")

	_, err = Grad(cost, model.Learnables()...)
	handleError(err, "Grad: ")

	var costVal, outVal8, outVal9 Value
	readCost := Read(cost, &costVal)
	readOut8 := Read(output[steps-2], &outVal8)
	readOut9 := Read(output[steps-1], &outVal9)
	WithName("readCost")(readCost)
	WithName("readOut8")(readOut8)
	WithName("readOut9")(readOut9)

	//g = g.SubgraphRoots(cost)
	prog, locMap, err := Compile(g)
	handleError(err, "Compile")
	vm := NewTapeMachine(g, WithPrecompiled(prog, locMap), BindDualValues(model.Learnables()...))

	// Now that we have our graph, program, and machine, we can start training
	solver := NewRMSPropSolver(WithLearnRate(learnrate), WithClip(clipVal), WithL2Reg(l2reg))

	start := time.Now()
	for i := 0; i < trainIter; i++ {
		xT, yT := GetTraningData()
		for i := 0; i < steps; i++ {
			Let(x[i], xT[i])
			Let(y[i], yT[i])
		}

		handleError(vm.RunAll(), "RunAll")
		// After running the machine, we want to update w and b
		solver.Step(model.LearnablesGrad())
		// move the pointer back to the beginning of the prog. Reset() does not delete any values
		vm.Reset()

		if i%viewIter == 0 {
			fmt.Printf("Interation #%v, Training cost: %#v\n", i, cost.Value())

			for i := 0; i < steps; i++ {
				fmt.Println("-xx------", i, "----xx------", x[i].Value())
				fmt.Println("-yy------", i, "----yy------", y[i].Value())
				fmt.Println("-out------", i, "----out------", output[i].Value())
			}
			fmt.Println("output8", "\n", outVal8)
			fmt.Println("output9", "\n", outVal9)
			fmt.Println("cost", "\n", costVal)
		}
	}
	fmt.Printf("Time taken: %v\n", time.Since(start))
	fmt.Println("cost", "\n", costVal)
	saver.Save(model)
}

// one hidden layer.
func NewRNNModel() (m *Model, err error) {
	var layer1, layer2, layer3 Layer
	if layer1, err = NewRNN("layer1", RNNOpts{
		InputSize:  2,
		HiddenSize: hiddenSize,
		//Activation: "Tanh",
	}); err != nil {
		return nil, err
	}

	if layer2, err = NewRNN("layer2", RNNOpts{
		InputSize:  hiddenSize,
		HiddenSize: hiddenSize,
		//Activation: "Tanh",
	}); err != nil {
		return nil, err
	}

	if layer3, err = NewFC("layer3", FCOpts{
		InputSize:  hiddenSize,
		OutputSize: 2,
		//Activation: "Linear",
	}); err != nil {
		return nil, err
	}

	m = NewModel(layer1, layer2, layer3)
	return
}

func NewGRUModel() (m *Model, err error) {
	var layer1, layer2, layer3 Layer
	if layer1, err = NewGRU("layer1", GRUOpts{
		InputSize:  2,
		HiddenSize: hiddenSize,
		//Activation: "Tanh",
	}); err != nil {
		return nil, err
	}

	if layer2, err = NewGRU("layer2", GRUOpts{
		InputSize:  hiddenSize,
		HiddenSize: hiddenSize,
		//Activation: "Tanh",
	}); err != nil {
		return nil, err
	}

	if layer3, err = NewFC("layer3", FCOpts{
		InputSize:  hiddenSize,
		OutputSize: 2,
		//Activation: "Linear",
	}); err != nil {
		return nil, err
	}

	m = NewModel(layer1, layer2, layer3)
	return
}

func NewLSTMModel() (m *Model, err error) {
	var layer1, layer2, layer3 Layer
	if layer1, err = NewLSTM("layer1", LSTMOpts{
		InputSize:  2,
		HiddenSize: hiddenSize,
		//Activation: "Tanh",
	}); err != nil {
		return nil, err
	}

	if layer2, err = NewLSTM("layer2", LSTMOpts{
		InputSize:  hiddenSize,
		HiddenSize: hiddenSize,
		//Activation: "Tanh",
	}); err != nil {
		return nil, err
	}

	if layer3, err = NewFC("layer3", FCOpts{
		InputSize:  hiddenSize,
		OutputSize: 2,
		//Activation: "Linear",
	}); err != nil {
		return nil, err
	}

	m = NewModel(layer1, layer2, layer3)
	return
}

// x2 = x1
func GetTraningData_Linear() (xT, yT []tensor.Tensor) {
	start := make([]float64, batchSize)
	foot := make([]float64, batchSize)
	for i := 0; i < batchSize; i++ {
		start[i] = rand.Float64() * 10
		foot[i] = rand.Float64()*10 + 0.01
	}

	for i := 0; i < steps; i++ {
		xb := make([]float64, 2*batchSize)
		yb := make([]float64, 2*batchSize)
		for j := 0; j < batchSize; j++ {
			xb[2*j] = foot[j]*float64(i) + start[j]
			xb[2*j+1] = xb[2*j]

			yb[2*j] = foot[j]*float64(i+1) + start[j]
			yb[2*j+1] = yb[2*j]
		}
		xT = append(xT, tensor.New(tensor.Of(tensor.Float64), tensor.WithShape(batchSize, 2), tensor.WithBacking(xb)))
		yT = append(yT, tensor.New(tensor.Of(tensor.Float64), tensor.WithShape(batchSize, 2), tensor.WithBacking(yb)))
	}
	return
}

func GetTraningData_ConstFoot() (xT, yT []tensor.Tensor) {
	start := make([]float64, batchSize)
	for i := 0; i < batchSize; i++ {
		start[i] = rand.Float64() * math.Pi
	}
	foot := rand.Float64()*math.Phi + 0.01

	for i := 0; i < steps; i++ {
		xb := make([]float64, 2*batchSize)
		yb := make([]float64, 2*batchSize)
		for j := 0; j < batchSize; j++ {
			xb[2*j] = foot*float64(i) + start[j]
			xb[2*j+1] = 10.0 * math.Sin(xb[2*j])

			yb[2*j] = foot*float64(i+1) + start[j]
			yb[2*j+1] = 10.0 * math.Sin(yb[2*j])
		}
		xT = append(xT, tensor.New(tensor.Of(tensor.Float64), tensor.WithShape(batchSize, 2), tensor.WithBacking(xb)))
		yT = append(yT, tensor.New(tensor.Of(tensor.Float64), tensor.WithShape(batchSize, 2), tensor.WithBacking(yb)))
	}
	return
}

func GetTraningData() (xT, yT []tensor.Tensor) {
	start := make([]float64, batchSize)
	foot := make([]float64, batchSize)
	for i := 0; i < batchSize; i++ {
		start[i] = rand.Float64() * math.Pi
		foot[i] = rand.Float64()*math.Phi + 0.01
	}

	for i := 0; i < steps; i++ {
		xb := make([]float64, 2*batchSize)
		yb := make([]float64, 2*batchSize)
		for j := 0; j < batchSize; j++ {
			xb[2*j] = foot[j]*float64(i) + start[j]
			xb[2*j+1] = 10.0 * math.Sin(xb[2*j])

			yb[2*j] = foot[j]*float64(i+1) + start[j]
			yb[2*j+1] = 10.0 * math.Sin(yb[2*j])
		}
		xT = append(xT, tensor.New(tensor.Of(tensor.Float64), tensor.WithShape(batchSize, 2), tensor.WithBacking(xb)))
		yT = append(yT, tensor.New(tensor.Of(tensor.Float64), tensor.WithShape(batchSize, 2), tensor.WithBacking(yb)))
	}
	return
}

func handleError(err error, s ...string) {
	if err != nil {
		if len(s) > 0 {
			log.Fatalln(s[0], err)
		} else {
			log.Fatalln(err)
		}
	}
}
