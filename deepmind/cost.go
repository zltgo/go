package deepmind

import (
	"github.com/pkg/errors"
	. "gorgonia.org/gorgonia"
)

type LossFunc func(output, target *Node) (cost *Node, err error)

// Add losses of every output
func Losses(outputs, targets Nodes, f LossFunc) (cost *Node, err error) {
	size := len(outputs)
	if size != len(targets) {
		return nil, errors.Errorf("mismatched sizes of outputs and targets: %v, %v", size, len(targets))
	}
	if size < 1 {
		return nil, errors.New("empty outputs and targets")
	}

	if size == 1 {
		cost, err = f(outputs[0], targets[0])
	} else {
		losses := make(Nodes, size)
		var loss *Node
		for i := 0; i < size; i++ {
			if loss, err = f(outputs[i], targets[i]); err != nil {
				return nil, err
			}
			losses[i] = loss
		}
		cost, err = ReduceAdd(losses)
	}

	if err == nil {
		WithName("cost")(cost)
	}
	return cost, err
}

// mean squared error
// The formula is as below:
//     MSE(y, y') = Mean{ (y - y')^2 }
func MeanSquared(output, target *Node) (*Node, error) {
	// output - target
	sub, err := Sub(output, target)
	if err != nil {
		return nil, err
	}
	sq, err := Square(sub)
	if err != nil {
		return nil, err
	}

	return Mean(sq)
}

// categorical cross entropy, this is the loss fucntion of choice for multi-class
// classification problems and softmax output units.
// The formula is as below:
// CCE(p, t) = -Mean{ t * log(p) }
func CrossEntropy(output, target *Node) (*Node, error) {
	// log(output), output cannot be zero, log(0) cause error
	logProb, err := Log(output)
	if err != nil {
		return nil, err
	}

	// target*log(output)
	hp, err := HadamardProd(target, logProb)
	if err != nil {
		return nil, err
	}

	// mean{i=0~n} hp
	mean, err := Mean(hp)
	if err != nil {
		return nil, err
	}

	//-sum
	return Neg(mean)
}

// BinaryXent is a convenience function for doing binary crossentropy stuff.
// This is the loss fucntion of choice for two-class classification problems
// and sigmoid output units.
// The formula is as below:
// BCE(p, t) = -Mean{ t * log(p) + (1 - t) * log(1-p)}
func BinaryCrossEntropy(output, target *Node) (*Node, error) {
	bx, err := BinaryXent(output, target)
	if err != nil {
		return nil, err
	}
	return Mean(bx)
}

// categorical cross entropy for one hot situation.
func OneHotCE(output *Node, targetId int) (*Node, error) {
	// log(output), output cannot be zero, log(0) cause error
	var logProb, target *Node
	var err error
	switch {
	case output.IsVec():
		target, err = Slice(output, S(targetId))
	case output.IsColVec():
		target, err = Slice(output, S(targetId), S(0))
	case output.IsRowVec():
		target, err = Slice(output, S(0), S(targetId))
	default:
		return nil, errors.Errorf("OneHotCE not yet implemented for shape%v", output.Shape())
	}
	if err != nil {
		return nil, err
	}

	if logProb, err = Log(target); err != nil {
		return nil, err
	}

	return Neg(logProb)
}

// Size of targets must equal to batch of output.
func OneHotCEBatch(output *Node, targetIds []int) (cost *Node, err error) {
	if !output.IsMatrix() {
		return nil, errors.Errorf("output expect to be a matrix, got dims: %v", output.Dims())
	}
	batchSize := output.Shape()[0]
	if batchSize != len(targetIds) {
		return nil, errors.Errorf("mismatched sizes of outputs and targetIds: %v, %v", batchSize, len(targetIds))
	}
	if batchSize == 0 {
		return nil, errors.New("empty outputs and targets")
	}

	var target, logProb *Node
	losses := make(Nodes, batchSize)
	for i := 0; i < batchSize; i++ {
		if target, err = Slice(output, S(i), S(targetIds[i])); err != nil {
			return nil, errors.Wrap(err, "Slice")
		}
		if logProb, err = Log(target); err != nil {
			return nil, err
		}
		if losses[i], err = Neg(logProb); err != nil {
			return nil, err
		}
	}
	cost, err = ReduceAdd(losses)
	if err == nil {
		WithName("cost")(cost)
	}
	return cost, err
}

//func (OneHotCE) getTargetIds(outputs, targets Nodes) ([]int, error) {
//	if len(targets) != 1 {
//		return nil, errors.Errorf("the size of targets supposed to be 1, got %v", len(targets))
//	}

//	//targetId must have int type
//	dt, err := DtypeOf(targets[0])
//	if err != nil {
//		return nil, err
//	}
//	if dt != tensor.Int {
//		return nil, errors.Errorf("type of targetId error, expected int, got %v", dt)
//	}

//	var ids []int
//	if ids, ok := targets[0].Value().Data().([]int); !ok {
//		return nil, errors.New("cannot get []int from target Node")
//	}

//	size := len(outputs)
//	if size != len(ids) {
//		return nil, errors.Errorf("mismatched sizes of outputs and targetId: %v, %v", size, len(ids))
//	}
//	if size < 1 {
//		return nil, errors.New("empty outputs and targets")
//	}
//	return ids, nil
//}
