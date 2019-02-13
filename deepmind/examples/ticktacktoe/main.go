package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
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
	viewIter   = 100
	hiddenSize = 30
	batchSize  = 100
)

func main() {
	rand.Seed(time.Now().Unix())

	alice := LoadOrCreateModel("./save/alice")
	bob := LoadOrCreateModel("./save/alice")

	ga := NewGraph()
	handleError(alice.Init(ga, tensor.Float64), "init model")
	gb := NewGraph()
	handleError(alice.Init(gb, tensor.Float64), "init model")

	// Now that we have our graph, program, and machine, we can start training
	solvera := NewRMSPropSolver(WithLearnRate(learnrate), WithClip(clipVal), WithL2Reg(l2reg))
	solverb := NewRMSPropSolver(WithLearnRate(learnrate), WithClip(clipVal), WithL2Reg(l2reg))

	start := time.Now()
	for i := 0; i < trainIter; i++ {
		sa := States{}
		sb := States{}

		retVal, err := train(input{ga, alice, solvera, sa}, input{ga, alice, solvera, sa})
		handleError(err, "Train")

		if i%viewIter == 0 {
			fmt.Printf("Interation #%v, Training retVal\n: %v", i, retVal)
		}
	}
	fmt.Printf("Time taken: %v\n", time.Since(start))
	SaveModel("./save/alice", alice)
	SaveModel("./save/bob", bob)
}

type retVal struct {
	costa  float64
	costb  float64
	stepsa []int
	stepsb []int
}

type input struct {
	g  *ExprGraph
	m  *Model
	sv Solver
	st States
}

func train(alice, bob input) (rv retVal, err error) {
	// random input value for alice
	// 000 000 000  alice
	// 000 000 000 bob
	// 000 010 000 last step of bob
	xb := make([]float64, 27)
	rd := rand.Intn(9)
	xb[18+rd] = 1.0
	xb := append(xb, xb...) //batch size = 2

	x := NewMatrix(alice.g, tensor.Float64, WithShape(2, 27), WithBacking(xb))
	for {
		output, err := alice.m.Forward(x, alice.st)
		aliceChess := Must(Slice(x, nil, S(0, 9)))
		remainChess := Must(OneSub(aliceChess))
		dot := Must(HadamardProd(remainChess, output))

	}

	return
}

func LoadOrCreateModel(path string) *Model {
	saver := NewJsonSaver(path)
	model, err := saver.Load()
	if os.IsNotExist(err) {
		model, err = NewLSTMModel()
	}
	handleError(err, "failed to load or create model: ")
	return model
}

func SaveModel(path string, m *Model) {
	saver := NewJsonSaver(path)
	saver.Save(m)
}

func NewLSTMModel() (m *Model, err error) {
	var layer1, layer2, layer3, layer4 Layer
	if layer1, err = NewLSTM("layer1", LSTMOpts{
		InputSize:  27,
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

	if layer3, err = NewLSTM("layer3", LSTMOpts{
		InputSize:  hiddenSize,
		HiddenSize: hiddenSize,
		//Activation: "Tanh",
	}); err != nil {
		return nil, err
	}

	if layer4, err = NewFC("layer4", FCOpts{
		InputSize:  hiddenSize,
		OutputSize: 9,
		Activation: "Sigmoid",
	}); err != nil {
		return nil, err
	}

	m = NewModel(layer1, layer2, layer3, layer4)
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
