package main

import (
	"fmt"

	"github.com/pkg/errors"
	. "gorgonia.org/gorgonia"
)

func main() {
	g := NewGraph()
	x := NewTensor(g, Float32, 3, WithName("x"), WithShape(2, 3, 3), WithInit(RangedFrom(0)))
	y := NewTensor(g, Float32, 3, WithName("y"), WithShape(2, 1, 3), WithInit(RangedFrom(0)))

	// it will not panic when I change the shape of y.
	// y := NewTensor(g, Float32, 3, WithName("y"), WithShape(2, 3, 2), WithInit(RangedFrom(0)))

	z, err := Concat(1, x, y)
	if err != nil {
		fmt.Println(errors.Wrap(err, "Concat"))
		return
	}

	Sum(z)

	m := NewLispMachine(g, ExecuteFwdOnly())
	if err = m.RunAll(); err != nil {
		fmt.Println(errors.Wrap(err, "run Concat"))
		return
	}
	fmt.Print("x: \n", x.Value())
	fmt.Print("y: \n", y.Value())

	//panic: runtime error: invalid memory address or nil pointer dereference
	//[signal SIGSEGV: segmentation violation code=0x1 addr=0x0 pc=0x885762]
	fmt.Print("Concat of x,y: \n", z.Value())
}
