package main

import (
	"fmt"

	"github.com/pkg/errors"
	. "gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

func runMul(xS, yS tensor.Shape) {
	g := NewGraph()
	x := NewMatrix(g, Float32, WithName("x"), WithShape(xS...), WithInit(RangedFrom(0)))
	y := NewMatrix(g, Float32, WithName("y"), WithShape(yS...), WithInit(RangedFrom(0)))

	z := Must(Mul(x, y))
	cost := Must(Sum(z))

	_, err := Grad(cost, x, y)
	if err != nil {
		fmt.Println(errors.Wrap(err, "Grad"))
		return
	}

	prog, locMap, err := Compile(g)
	if err != nil {
		fmt.Println(errors.Wrap(err, "Compile"))
		return
	}
	m := NewTapeMachine(g, WithPrecompiled(prog, locMap))
	if err = m.RunAll(); err != nil {
		fmt.Println(errors.Wrap(err, "RunAll"))
		return
	}
	fmt.Println("cost:", cost.Value())
}

func main2() {
	//it works, cost = 22
	runMul(tensor.Shape{2, 3}, tensor.Shape{3, 2})

	//panic: Node Σ[0](%2) :: float32, has 0 dimensions(Shape: ()). Input shape is (1, 1), which has 2 dimensions
	runMul(tensor.Shape{2, 2}, tensor.Shape{2, 1})

	//panic: Node Σ[1](%2) :: float32, has 0 dimensions(Shape: ()). Input shape is (1, 1), which has 2 dimensions
	runMul(tensor.Shape{1, 2}, tensor.Shape{2, 2})
}
