package main

import (
	"fmt"
	"log"
	"os"
	"time"

	. "github.com/zltgo/deepmind"
	. "gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

var (
	// gradient update stuff
	l2reg     = 0.000001
	learnrate = 0.01
	clipVal   = 5.0
	trainIter = 1000
	viewIter  = 100
	batchSize = 1
)

func main() {
	saver := NewJsonSaver("./save")
	model, err := saver.Load()
	if os.IsNotExist(err) {
		model, err = NewXorModel()
	}
	handleError(err, "failed to load or create model: ")

	g := NewGraph()
	handleError(model.Init(g, tensor.Float32), "init model")

	// input size = 2, must give a name, bug?
	x := NewMatrix(g, tensor.Float32, WithShape(batchSize, 2), WithName("x"))
	// output size = 2
	y := NewMatrix(g, tensor.Float32, WithShape(batchSize, 2), WithName("y"))

	output, err := model.Forward(x, nil)
	handleError(err, "model.Forward: ")

	cost, err := BinaryCrossEntropy(output, y)
	handleError(err, "BinaryCrossEntropy: ")

	_, err = Grad(cost, model.Learnables()...)
	handleError(err, "Grad: ")

	var costVal, outVal Value
	readCost := Read(cost, &costVal)
	readOut := Read(output, &outVal)
	WithName("readCost")(readCost)
	WithName("readOut")(readOut)

	//g = g.SubgraphRoots(cost)
	prog, locMap, err := Compile(g)
	handleError(err, "Compile")
	vm := NewTapeMachine(g, WithPrecompiled(prog, locMap), BindDualValues(model.Learnables()...))

	// Now that we have our graph, program, and machine, we can start training
	solver := NewRMSPropSolver(WithLearnRate(learnrate), WithL2Reg(l2reg), WithClip(clipVal))
	start := time.Now()
	for i := 0; i < trainIter; i++ {
		xT, yT := GetTraningData(batchSize)
		Let(x, xT)
		Let(y, yT)

		handleError(vm.RunAll(), "RunAll")
		// After running the machine, we want to update w and b
		solver.Step(model.Learnables())
		// move the pointer back to the beginning of the prog. Reset() does not delete any values
		vm.Reset()

		if i%viewIter == 0 {
			fmt.Printf("Interation #%v, Training cost: %#v\n", i, cost.Value())
			fmt.Println(x.Name(), "\n", x.Value())
			fmt.Println(y.Name(), "\n", y.Value())
			fmt.Println("output", "\n", outVal)
			fmt.Println("cost", "\n", costVal)
		}
	}
	fmt.Printf("Time taken: %v\n", time.Since(start))
	for _, wb := range model.Learnables() {
		fmt.Println(wb.Name(), "\n", wb.Value())
	}
	saver.Save(model)
}

// one hidden layer with 4 neurons.
func NewXorModel() (m *Model, err error) {
	var layer1, layer2 Layer
	if layer1, err = NewFC("layer1", FCOpts{
		InputSize:  2,
		OutputSize: 4,
		Activation: "ReLU",
	}); err != nil {
		return nil, err
	}

	if layer2, err = NewFC("layer2", FCOpts{
		InputSize:  4,
		OutputSize: 2,
		Activation: "Sigmoid",
	}); err != nil {
		return nil, err
	}

	m = NewModel(layer1, layer2)
	return
}

func GetTraningData(batchSize int) (xT, yT tensor.Tensor) {
	xb := Uniform32(-10.0, 10.0, batchSize, 2)
	yb := make([]float32, batchSize*2)
	for i := 0; i < batchSize*2; i += 2 {
		// one > 0 and other < 0, yb = 1
		if xb[i]*xb[i+1] < 0 {
			yb[i] = 0.0
			yb[i+1] = 1.0
		} else {
			yb[i] = 1.0
			yb[i+1] = 0
		}
	}
	xT = tensor.New(tensor.Of(tensor.Float32), tensor.WithShape(batchSize, 2), tensor.WithBacking(xb))
	yT = tensor.New(tensor.Of(tensor.Float32), tensor.WithShape(batchSize, 2), tensor.WithBacking(yb))
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
