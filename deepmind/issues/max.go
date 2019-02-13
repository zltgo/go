package main

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/zltgo/deepmind"
	. "gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

func main() {
	g := NewGraph()

	xt := tensor.New(tensor.Of(tensor.Float32), tensor.WithShape(5), tensor.WithBacking([]float32{0.1, 0.2, 0.3, 0.3, 0.1}))
	x := NewVector(g, Float32, WithValue(xt))

	max := Must(Max(x))
	z := Must(Gte(x, max, true))

	m := NewLispMachine(g, ExecuteFwdOnly(), WithManualGradient())
	if err := m.RunAll(); err != nil {
		//RunAll: Running Node: >= true(%0, %1) :: Vector float32: Failed to execute MaxAlong[0] in node MaxAlong[0](%0) :: float32: Failed to carry op.Do(): maxOp.
		fmt.Println(errors.Wrap(err, "RunAll"))
		return
	}
	fmt.Println("max of x: %v\n", z.Value())
}

func main_StableSoftMax() {
	g := NewGraph()
	x := NewVector(g, Float64, WithName("x"), WithShape(5), deepmind.WithBacking([]float64{1.1, 1.2, 1.3, 1.3, 1.2}))
	ssm := Must(StableSoftMax(x))

	m := NewLispMachine(g, ExecuteFwdOnly(), WithManualGradient())
	if err := m.RunAll(); err != nil {
		fmt.Println(errors.Wrap(err, "RunAll"))
		return
	}
	//panic: Operation failed: Failed to infer shape. Op: Σ[1]: Shape mismatch: along is [1]. Shape is (5)
	fmt.Println("StableSoftMax of x: %v\n", ssm.Value())
}
