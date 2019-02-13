package main

import (
	"bytes"
	"fmt"
	"reflect"
	"runtime"

	"math"

	"github.com/chewxy/hm"
	"github.com/pkg/errors"
	. "github.com/zltgo/deepmind"
	"gonum.org/v1/gonum/mat"
	. "gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

func example_DropOut() {
	g := NewGraph()
	x := nodeFromAny(g, []float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, "x")
	dropx, err := Dropout(x, 0.2)
	if err != nil {
		fmt.Println(errors.Wrap(err, "dropout"))
		return
	}

	// 必须有ExecuteFwdOnly()，否则只能使用TapeMachine
	m := NewLispMachine(g, ExecuteFwdOnly())
	if err = m.RunAll(); err != nil {
		fmt.Println(errors.Wrap(err, "run dropout"))
		return
	}
	fmt.Println("dropout of x: ", dropx.Value())
	// 要查一下为什么最后要乘以prob作为最终输出
	// dropx: [0.2  0.4  0.6  0.8  ... 1.4  1.6    0    2]
}

func example_Sigmoid() {
	g := NewGraph()

	x := NewTensor(g, tensor.Float32, 2, WithShape(2, 10), WithBacking([]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 1, 2, 3, 4, 5, -1, -10, -100, -1000, -2000}))
	sx, err := Sigmoid(x)
	if err != nil {
		fmt.Println(errors.Wrap(err, "Sigmoid"))
		return
	}

	// 必须有ExecuteFwdOnly()，否则只能使用TapeMachine
	m := NewLispMachine(g, ExecuteFwdOnly())
	if err = m.RunAll(); err != nil {
		fmt.Println(errors.Wrap(err, "run softmax"))
		return
	}
	fmt.Println("Sigmoid of x: ", sx.Value())
	//softmax of x:  [  7.80134e-05  0.00021206241  0.00057644537   0.0015669409  ...    0.03147285     0.08555202     0.23255466      0.6321494]
}

func example_SoftMax() {
	g := NewGraph()
	x := nodeFromAny(g, []float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, "x")
	y := NewTensor(g, tensor.Float32, 2, WithShape(2, 10), WithBacking([]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 1, 2, 3, 4, 5, 6, 7, 8, 9, -20}))
	sx, err := SoftMax(x)
	if err != nil {
		fmt.Println(errors.Wrap(err, "softmax"))
		return
	}

	sy, err := SoftMax(y)
	if err != nil {
		fmt.Println(errors.Wrap(err, "softmax"))
		return
	}

	// 必须有ExecuteFwdOnly()，否则只能使用TapeMachine
	m := NewLispMachine(g, ExecuteFwdOnly())
	if err = m.RunAll(); err != nil {
		fmt.Println(errors.Wrap(err, "run softmax"))
		return
	}
	fmt.Println("softmax of x: ", sx.Value())
	//softmax of x:  [  7.80134e-05  0.00021206241  0.00057644537   0.0015669409  ...    0.03147285     0.08555202     0.23255466      0.6321494]
	fmt.Println("softmax of y: ", sy.Value())
}

func example_ConCat() {
	g := NewGraph()

	x := NewTensor(g, Float32, 3, WithName("x"), WithShape(2, 3, 3), WithInit(RangedFrom(0)))
	y := NewTensor(g, Float32, 3, WithName("y"), WithShape(2, 3, 2), WithInit(RangedFrom(0)))
	//bug
	//y := NewTensor(g, Float32, 3, WithName("y"), WithShape(2, 3, 1), WithInit(RangedFrom(0)))
	z, err := Concat(2, x, y)
	if err != nil {
		fmt.Println(errors.Wrap(err, "Concat"))
		return
	}

	// 必须有ExecuteFwdOnly()，否则只能使用TapeMachine
	m := NewLispMachine(g, ExecuteFwdOnly())
	if err = m.RunAll(); err != nil {
		fmt.Println(errors.Wrap(err, "run Concat"))
		return
	}
	fmt.Print("x: \n", x.Value())
	fmt.Print("y: \n", y.Value())
	fmt.Print("Concat of x,y: \n", z.Value())
}

// 逐个求倒数
func example_Inverse() {
	g := NewGraph()
	xt := tensor.New(tensor.Of(tensor.Float32), tensor.WithShape(2, 3, 2))
	xt.Memset(float32(2))
	x := NewTensor(g, Float32, xt.Dims(), WithName("x"), WithValue(xt))

	y, err := Inverse(x)
	if err != nil {
		fmt.Println(errors.Wrap(err, "Inverse"))
		return
	}

	// 必须有ExecuteFwdOnly()，否则只能使用TapeMachine
	m := NewLispMachine(g, ExecuteFwdOnly())
	if err = m.RunAll(); err != nil {
		fmt.Println(errors.Wrap(err, "run Inverse"))
		return
	}
	fmt.Print("Inverse of x: \n", y.Value())
	//Inverse of x:
	//⎡0.5  0.5⎤
	//⎢0.5  0.5⎥
	//⎣0.5  0.5⎦

	//⎡0.5  0.5⎤
	//⎢0.5  0.5⎥
	//⎣0.5  0.5⎦
}

// bug?? OneHotVector的输入参数ID并没有起作用
func example_OneHotVector() {
	x := OneHotVector(3, 4, Int)
	fmt.Println("OneHotVector of 4/3: ", x.Value())
	//OneHotVector of 4/3:  [1  1  1  1]
}

//log以自然数为底
func example_Log() {
	g := NewGraph()
	x := NewConstant(10.0, WithName("x"), In(g))
	logx, err := Log(x)
	if err != nil {
		fmt.Println(errors.Wrap(err, "Log"))
		return
	}

	log1px, err := Log1p(x)
	if err != nil {
		fmt.Println(errors.Wrap(err, "Log1p"))
		return
	}

	log2x, err := Log2(x)
	if err != nil {
		fmt.Println(errors.Wrap(err, "Log2"))
		return
	}

	// 必须有ExecuteFwdOnly()，否则只能使用TapeMachine
	m := NewLispMachine(g, ExecuteFwdOnly())
	if err = m.RunAll(); err != nil {
		fmt.Println(errors.Wrap(err, "run Inverse"))
		return
	}

	fmt.Println("logx: ", logx.Value(), "    math:", math.Log(math.E))
	fmt.Println("log1px: ", log1px.Value(), "    math:", math.Log1p(math.E))
	fmt.Println("log2x: ", log2x.Value(), "    math:", math.Log2(2.0))
	//logx:  2.302585092994046     math: 2.302585092994046
	//log1px:  2.3978952727983707     math: 2.3978952727983707
	//log2x:  3.321928094887362     math: 3.321928094887362
}

//把两个向量按元素相乘组成一个矩阵
//[a0*b0 a0*b1 ... a0*bn]
//[a1*b0 a1*b1 ... a1*bn]
//[...                               ]
//[...       ...                     ]
//[an*b0 an*b1 ... an*bn]
//相当于把n*1的列向量与1*n的行向量相乘
func example_OuterProd() {
	g := NewGraph()
	x := NewVector(g, Float32, WithName("x"), WithShape(5), WithInit(RangedFrom(0)))
	y := NewVector(g, Float32, WithName("y"), WithShape(5), WithInit(RangedFrom(1)))
	z, err := OuterProd(x, y)
	if err != nil {
		fmt.Println(errors.Wrap(err, "OuterProd"))
		return
	}

	// 必须有ExecuteFwdOnly()，否则只能使用TapeMachine
	m := NewLispMachine(g, ExecuteFwdOnly())
	if err = m.RunAll(); err != nil {
		fmt.Println(errors.Wrap(err, "run OuterProd"))
		return
	}
	fmt.Println("x: ", x.Value())
	fmt.Println("y: ", y.Value())
	fmt.Print("OuterProd of x,y: \n", z.Value())

	//矩阵乘法作为对比
	a := mat.NewDense(5, 1, []float64{0, 1, 2, 3, 4})
	b := mat.NewDense(1, 5, []float64{1, 2, 3, 4, 5})
	var zm mat.Dense
	zm.Mul(a, b)
	r, c := zm.Dims()
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			fmt.Print(zm.At(i, j), " ")
		}
		fmt.Print("\n")
	}
}

func example_Slice() {
	g := NewGraph()
	x := NewTensor(g, Float32, 3, WithName("x"), WithShape(2, 3, 4), WithInit(RangedFrom(0)))
	xaa0 := Must(Slice(x, nil, nil, S(0)))
	xaa0_1 := Must(Slice(x, nil, nil, S(0, 2)))
	x0aa := Must(Slice(x, S(0)))
	xa0a := Must(Slice(x, nil, S(0)))
	x000 := Must(Slice(x, S(0), S(0), S(0)))

	//y := NewMatrix(g, tensor.Float32, WithShape(3, 1), WithInit(RangedFrom(0)))
	//y0 := Must(Slice(y, nil, S(0)))

	//看看会不会改变原值
	if err := Let(x000, 8); err != nil {
		fmt.Println(errors.Wrap(err, "let slice"))
	}

	m := NewLispMachine(g, ExecuteFwdOnly())
	if err := m.RunAll(); err != nil {
		fmt.Println(errors.Wrap(err, "run Slice"))
		return
	}

	fmt.Print("x:\n", x.Value())
	fmt.Println("x000:", x000.Value())
	fmt.Print("xaa0:\n", xaa0.Value())
	fmt.Print("xaa0_1:\n", xaa0_1.Value())
	fmt.Print("x0aa:\n", x0aa.Value())
	fmt.Print("xa0a:\n", xa0a.Value())
	//fmt.Print("y0:\n", y0.Value())
}

// Gorgonia might delete values from nodes so we are going to save it
// and print it out later
func example_Read() {
	g := NewGraph()
	x := NewTensor(g, Float32, 2, WithName("x"), WithShape(2, 3), WithInit(RangedFrom(0)))
	var yV Value
	// z并不保存x的值，其作用是作为一个节点来控制是否执行Read操作
	z := Read(x, &yV)

	//默认z节点在g中
	m := NewLispMachine(g, ExecuteFwdOnly())
	if err := m.RunAll(); err != nil {
		fmt.Println(errors.Wrap(err, "run Read1"))
		return
	}

	fmt.Print("x:\n", x.Value())
	fmt.Print("y:\n", yV)
	fmt.Println("z:", z.Value())

	//去掉z节点
	g = g.SubgraphRoots(x)
	Let(x, tensor.NewDense(tensor.Float32, tensor.Shape{2, 3}, tensor.WithBacking(tensor.Range(tensor.Float32, 6, 12))))
	m = NewLispMachine(g, ExecuteFwdOnly())
	if err := m.RunAll(); err != nil {
		fmt.Println(errors.Wrap(err, "run Read2"))
		return
	}

	fmt.Print("x:\n", x.Value())
	fmt.Print("y:\n", yV)
}

func example_Mul() {
	g := NewGraph()

	x := NewTensor(g, Float32, 2, WithName("x"), WithShape(1, 2), WithInit(RangedFrom(0)))
	//y := NewVector(g, Float32, WithName("y"), WithShape(2), WithInit(RangedFrom(0)))
	y := NewMatrix(g, Float32, WithName("y"), WithShape(2, 1), WithInit(RangedFrom(0)))

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
		fmt.Println(errors.Wrap(err, "run Grad"))
		return
	}

	fmt.Println("x: \n", x.Value())
	fmt.Println("y: \n", y.Value())
	fmt.Println("Mul of x,y: \n", z.Value())
	fmt.Println("cost: \n", cost.Value())
}

// x and y cannot use the same value
func example_Clone() {
	g := NewGraph()
	h := NewGraph()

	xt := tensor.New(tensor.Of(tensor.Float32), tensor.WithShape(2, 3))
	xt.Memset(float32(2))
	x := NewTensor(g, Float32, xt.Dims(), WithName("x"), WithValue(xt))
	y := NewTensor(g, Float32, xt.Dims(), WithName("y"), WithValue(xt))
	z := x.CloneTo(h)

	xt.Memset(float32(3))
	fmt.Print("x: \n", x.Value())
	fmt.Print("y: \n", y.Value())
	fmt.Print("z: \n", z.Value())
}

func example_WriteCsv() {
	t := tensor.NewDense(tensor.Float32, tensor.Shape{2, 3, 2}, tensor.WithBacking(tensor.Range(tensor.Float32, 0, 12)))

	buf := new(bytes.Buffer)
	err := t.WriteCSV(buf)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(buf.String())
	}

	err = t.WriteNpy(buf)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(buf.String())
	}
}

func example_Shape() {
	g := NewGraph()

	x1T := tensor.New(tensor.WithShape(1), tensor.WithBacking([]int{1}))
	x11T := tensor.New(tensor.WithShape(1, 1), tensor.WithBacking([]int{1}))
	x3T := tensor.New(tensor.WithShape(3), tensor.WithBacking([]int{1, 2, 3}))
	x13T := tensor.New(tensor.WithShape(1, 3), tensor.WithBacking([]int{1, 2, 3}))
	x31T := tensor.New(tensor.WithShape(3, 1), tensor.WithBacking([]int{1, 2, 3}))

	c := NodeFromAny(g, 0.3)
	x1 := NodeFromAny(g, x1T)
	x11 := NodeFromAny(g, x11T)
	x3 := NodeFromAny(g, x3T)
	x13 := NodeFromAny(g, x13T)
	x31 := NodeFromAny(g, x31T)
	b := NewTensor(g, tensor.Float32, 0)

	fmt.Printf("c: isMatrix %v, isCol %v, isRow %v, isVec %v, IsScalar %v, dims:%v\n", c.IsMatrix(), c.IsColVec(), c.IsRowVec(), c.IsVector(), c.IsScalar(), c.Dims())
	fmt.Printf("b: isMatrix %v, isCol %v, isRow %v, isVec %v, IsScalar %v, dims:%v\n", b.IsMatrix(), b.IsColVec(), b.IsRowVec(), b.IsVector(), b.IsScalar(), b.Dims())

	fmt.Printf("x1T: isMatrix %v, isCol %v, isRow %v, isVec %v, IsScalar %v\n", x1T.IsMatrix(), x1T.IsColVec(), x1T.IsRowVec(), x1T.IsVector(), x1T.IsScalar())
	fmt.Printf("x11T: isMatrix %v, isCol %v, isRow %v, isVec %v, IsScalar %v\n", x11T.IsMatrix(), x11T.IsColVec(), x11T.IsRowVec(), x11T.IsVector(), x11T.IsScalar())
	fmt.Printf("x3T: isMatrix %v, isCol %v, isRow %v, isVec %v\n", x3T.IsMatrix(), x3T.IsColVec(), x3T.IsRowVec(), x3T.IsVector())
	fmt.Printf("x13T: isMatrix %v, isCol %v, isRow %v, isVec %v\n", x13T.IsMatrix(), x13T.IsColVec(), x13T.IsRowVec(), x13T.IsVector())
	fmt.Printf("x31T: isMatrix %v, isCol %v, isRow %v, isVec %v\n\n", x31T.IsMatrix(), x31T.IsColVec(), x31T.IsRowVec(), x31T.IsVector())

	fmt.Printf("x1: isMatrix %v, isCol %v, isRow %v, isVec %v, IsScalar %v\n", x1.IsMatrix(), x1.IsColVec(), x1.IsRowVec(), x1.IsVector(), x1.IsScalar())
	fmt.Printf("x11: isMatrix %v, isCol %v, isRow %v, isVec %v, IsScalar %v\n", x11.IsMatrix(), x11.IsColVec(), x11.IsRowVec(), x11.IsVector(), x11.IsScalar())
	fmt.Printf("x3: isMatrix %v, isCol %v, isRow %v, isVec %v\n", x3.IsMatrix(), x3.IsColVec(), x3.IsRowVec(), x3.IsVector())
	fmt.Printf("x13: isMatrix %v, isCol %v, isRow %v, isVec %v\n", x13.IsMatrix(), x13.IsColVec(), x13.IsRowVec(), x13.IsVector())
	fmt.Printf("x31: isMatrix %v, isCol %v, isRow %v, isVec %v\n", x31.IsMatrix(), x31.IsColVec(), x31.IsRowVec(), x31.IsVector())
}

func printValue(name string, v *Node) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(name, r)
		}
	}()

	fmt.Printf("%s:\n%v\n", name, v.Value())
}
func example_Reshape() {
	g := NewGraph()

	x := NewTensor(g, Float32, 2, WithName("x"), WithShape(2, 4), WithInit(RangedFrom(0)))
	fmt.Print("x:\n", x.Value())
	xxx, _ := Concat(0, x, x)
	x224, err224 := Reshape(xxx, tensor.Shape{2, 2, 4})

	x42, err42 := Reshape(x, tensor.Shape{4, 2})
	x33, err33 := Reshape(x, tensor.Shape{3, 3})
	x124, err124 := Reshape(x, tensor.Shape{1, 2, 4})

	// 必须有ExecuteFwdOnly()，否则只能使用TapeMachine
	m := NewLispMachine(g, ExecuteFwdOnly())
	if err := m.RunAll(); err != nil {
		fmt.Println(errors.Wrap(err, "run Reshape"))
		return
	}

	if err42 != nil {
		fmt.Println(errors.Wrap(err42, "reshape 42"))
	} else {
		printValue("x42", x42)
	}

	if err33 != nil {
		fmt.Println(errors.Wrap(err33, "reshape 33"))
	} else {
		printValue("x33", x33) //bug
	}

	if err124 != nil {
		fmt.Println(errors.Wrap(err124, "reshape 124"))
	} else {
		printValue("x124", x124)
	}

	if err224 != nil {
		fmt.Println(errors.Wrap(err224, "reshape 224"))
	} else {
		printValue("x224", x224)
	}
}

func dtypeOf(t hm.Type) (retVal tensor.Dtype, err error) {
	switch p := t.(type) {
	case tensor.Dtype:
		retVal = p
	case TensorType:
		return dtypeOf(p.Of)
	case hm.TypeVariable:
		err = errors.Errorf("instance %v does not have a dtype", p)
	default:
		err = errors.Errorf("not yet implemented for %v", p)
	}
	return
}

func example_DtypeOf() {
	g := NewGraph()
	x := NewTensor(g, tensor.Float32, 2, WithName("x"), WithShape(2, 4), WithInit(RangedFrom(0)))
	dt, err := DtypeOf(x)

	fmt.Println(dt, err, dt == Float32)

	b := tensor.New(tensor.Of(tensor.Float32), tensor.WithShape(2, 3))
	fmt.Println(b)
}

func example_Add() {
	g := NewGraph()

	x := NewMatrix(g, tensor.Float32, WithShape(1, 3), WithInit(RangedFrom(0)))
	//y := NewVector(g, tensor.Float32, WithShape(1), WithInit(RangedFrom(1)))
	y := NewScalar(g, tensor.Float32, WithValue(float32(0)))

	z, err := Add(x, y)
	if err != nil {
		fmt.Println(errors.Wrap(err, "Add"))
		return
	}

	// 必须有ExecuteFwdOnly()，否则只能使用TapeMachine
	m := NewLispMachine(g, ExecuteFwdOnly())
	if err = m.RunAll(); err != nil {
		fmt.Println(errors.Wrap(err, "run Add"))
		return
	}
	fmt.Println("x: \n", x.Value())
	fmt.Println("y: \n", y.Value())
	fmt.Println("Add of x,y: \n", z.Value())
}

func example_Grad() {
	g := NewGraph()

	x := NewMatrix(g, tensor.Float32, WithShape(2, 3), WithInit(RangedFrom(0)), WithName("x"))
	//	//w := NewVector(g, tensor.Float32, WithShape(3), WithInit(RangedFrom(0)), WithName("w"))
	w := NewMatrix(g, tensor.Float32, WithShape(3, 2), WithInit(RangedFrom(0)), WithName("w"))
	y := NewMatrix(g, tensor.Float32, WithShape(2, 2), WithInit(RangedFrom(0)), WithName("y"))
	//b := NewVector(g, tensor.Float32, WithShape(2), WithInit(RangedFrom(0)), WithName("b"))
	b := NewScalar(g, tensor.Float32, WithValue(float32(0.0)), WithName("_b"))

	z, err := Fxwb(Activations.Get("Sigmoid"), x, w, b)
	if err != nil {
		fmt.Println(errors.Wrap(err, "Fwxb"))
		return
	}

	//	z, err = Reshape(z, tensor.Shape{2})
	//	if err != nil {
	//		fmt.Println(errors.Wrap(err, "Reshape"))
	//		return
	//	}

	//y := NewVector(g, tensor.Float32, WithShape(2), WithInit(RangedFrom(0)))
	cost, err := MeanSquared(z, y)
	if err != nil {
		fmt.Println(errors.Wrap(err, "MeanSquared"))
		return
	}

	grads, err := Grad(cost, w, b)
	if err != nil {
		fmt.Println(errors.Wrap(err, "Grad"))
		return
	}

	prog, locMap, err := Compile(g)
	if err != nil {
		fmt.Println(errors.Wrap(err, "CompileFunction"))
		return
	}
	m := NewTapeMachine(g, WithPrecompiled(prog, locMap))

	if err = m.RunAll(); err != nil {
		fmt.Println(errors.Wrap(err, "run Grad"))
		return
	}
	fmt.Println("grad_w: \n", grads[0].Value())
	fmt.Println("grad_b: \n", grads[1].Value())
	fmt.Println("cost: \n", cost.Value())
}

func example_Tensordot() {
	g := NewGraph()

	x := NewTensor(g, Float32, 2, WithName("x"), WithShape(2, 1), WithBacking([]float64{2.0, 1.0}))
	y := NewTensor(g, Float32, 2, WithName("x"), WithShape(2, 3), WithInit(RangedFrom(0)))

	z := Must(Tensordot([]int{0}, []int{0}, x, y))

	// 必须有ExecuteFwdOnly()，否则只能使用TapeMachine
	m := NewLispMachine(g, ExecuteFwdOnly())
	if err := m.RunAll(); err != nil {
		fmt.Println(errors.Wrap(err, "run Add"))
		return
	}

	fmt.Println("x: \n", x.Value())
	fmt.Println("y: \n", y.Value())
	fmt.Println("Tensordot of x,y: \n", z.Value())
}

func example_GTe() {
	g := NewGraph()
	t := tensor.New(tensor.WithShape(5), tensor.WithBacking([]float64{0.1, 0.2, 0.3, 0.4, 0.5}))
	rv, err := t.Max(0)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(rv.Data())
	}

	t2, err := t.Gte(rv)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(t2.Data())
	}

	x := NewTensor(g, Float64, 2, WithName("x"), WithShape(2, 5), WithBacking([]float64{1.0, 2.0, 3.0, 4.0, 5.0, 1.0, 2.0, 3.0, 4.0, 5.0}))
	//y := NewScalar(g, Float64, WithValue(3.0))

	z := Must(Max(x))
	//x = Must(Sub(x, z))
	//	if _, err := Sum(z); err != nil {
	//		fmt.Println(errors.Wrap(err, "Sum"))
	//		return
	//	}
	//c := Must(Gte(x, y, true))

	// 必须有ExecuteFwdOnly()，否则只能使用TapeMachine
	m := NewLispMachine(g, ExecuteFwdOnly(), WithManualGradient())
	if err := m.RunAll(); err != nil {
		fmt.Println(errors.Wrap(err, "run Gte"))
		return
	}

	fmt.Println("x: \n", x.Value())
	//	fmt.Println("y: \n", y.Value())
	fmt.Println("max of x,y: \n", z.Value())
	//fmt.Println("gte: \n", c.Value())
}

//basic usage
func main() {
	//	run(example_DropOut)
	//run(example_SoftMax)
	//  run(example_ConCat)
	//	run(example_Inverse)
	//	run(example_OneHotVector)
	//	run(example_Log)
	//	run(example_OuterProd)
	//run(example_Slice)
	// run(example_Read)
	//run(example_Mul)
	//	run(example_Clone)
	//	run(example_WriteCsv)
	//  run(example_Shape)
	//   run(example_Reshape)
	//   run(example_DtypeOf)
	// run(example_Add)
	//	run(example_Grad)
	// run(example_Sigmoid)
	//run(example_Tensordot)
	run(example_GTe)
}

func run(f func()) {
	fmt.Println("---------------", runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name(), "---------------")
	f()
}

func nodeFromAny(g *ExprGraph, any interface{}, name string) *Node {
	if reflect.TypeOf(any).Kind() == reflect.Slice {
		v := tensor.New(tensor.WithBacking(any))
		return NodeFromAny(g, v, WithName(name))
	}
	return NodeFromAny(g, any, WithName(name))
}
