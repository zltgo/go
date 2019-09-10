package main

import (
	"fmt"

	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"github.com/tensorflow/tensorflow/tensorflow/go/op"
)

func main() {
	// 这里，我们打算要: 创建图

	// 我们要定义两个占位符用于在运行的时候传入
	// 第一个占位符 A 将是一个 [2, 2] 的整数张量
	// 第二个占位符 x 将是一个 [2, 1] 的整数张量

	// 然后，我们要计算 Y = Ax

	// 创建图的第一个节点： 一个空的节点，位于图的根
	root := op.NewScope()

	// 定义两个占位符
	A := op.Placeholder(root.SubScope("input"), tf.Int32, op.PlaceholderShape(tf.MakeShape(2, 2)))
	x := op.Placeholder(root.SubScope("input"), tf.Int32, op.PlaceholderShape(tf.MakeShape(2, 1)))

	// 定义接收A和x作为输入参数的操作节点
	product := op.MatMul(root, A, x)
	//op.BiasAdd()
	//op.Abs()

	// 每次我们把一个`Scope`穿给操作符的时候，把操作放在作用域的下面。
	// 这样，我们有了一个空的作用域（由NewScope创建）：空的作用域是图的根，因此可以用“/”来表示。

	// 现在，我们让tensorflow按照我们的定义来创建图。
	// 把作用域和OP结合起来，创建具体的图。

	graph, err := root.Finalize()
	if err != nil {
		// 这里没办法处理这个错误：
		// 如果我们错误的定义了图，我们必须手工修改这个定义。

		// 这就跟SQL查询一样：如果查询语句在语法上有问题，我们只能重新写
		panic(err.Error())
	}

	// 如果在这里，图在语法上是正确的。
	// 我们就可以把它放到一个Session里，并执行。

	var sess *tf.Session
	sess, err = tf.NewSession(graph, &tf.SessionOptions{})
	if err != nil {
		panic(err.Error())
	}

	// 要使用占位符，我们必须创建一个Tensors，这个Tensors包含要反馈到网络的数值
	var matrix, column *tf.Tensor

	// A = [ [1, 2], [-1, -2] ]
	if matrix, err = tf.NewTensor([2][2]int32{{1, 2}, {-1, -2}}); err != nil {
		panic(err.Error())
	}
	// x = [ [10], [100] ]
	if column, err = tf.NewTensor([2][1]int32{{10}, {100}}); err != nil {
		panic(err.Error())
	}

	var results []*tf.Tensor
	if results, err = sess.Run(map[tf.Output]*tf.Tensor{
		A: matrix,
		x: column,
	}, []tf.Output{product}, nil); err != nil {
		panic(err.Error())
	}
	for _, result := range results {
		fmt.Println(result.Value().([][]int32))
	}
}
