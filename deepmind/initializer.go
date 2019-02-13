package deepmind

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	. "gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

var (
	ErrorEmptyInitializer = errors.New("the initializer string is empty")

	parameterFail = "%s expected %v paramer, got %v"
	nyiTypeFail   = "%s not yet implemented for %v"

	// cache all the initializer functions
	Initializers initializerMap
)

type Initializer func(opts []float64) (InitWFn, error)

func init() {
	Initializers = make(map[string]Initializer, 9)

	Initializers.Register("Binomial", binomial)
	Initializers.Register("GlorotU", glorotU)
	Initializers.Register("GlorotN", glorotN)
	Initializers.Register("Uniform", uniform)
	Initializers.Register("HeEtAlN", heEtAlN)
	Initializers.Register("HeEtAlU", heEtAlU)
	Initializers.Register("Gaussian", gaussian)
	Initializers.Register("RangedFrom", rangedFrom)
	Initializers.Register("MemSet", memSet)
	Initializers.Register("MemCopy", memCopy)
}

// GetInitWFn gets InitWFn from a string like "Gaussian(0, 0.08)"
func GetInitWFn(str string) (InitWFn, error) {
	name, opts, err := parseInitStr(str)
	if err != nil {
		return nil, err
	}
	ir := Initializers.Get(name)
	if ir == nil {
		return nil, errors.New("unknown initializer name:" + name)
	}
	return ir(opts)
}

func DefaultGetInitWFn(master, slaver string) (InitWFn, error) {
	if strings.TrimSpace(master) != "" {
		return GetInitWFn(master)
	}
	return GetInitWFn(slaver)
}

//Cache all the activations
type initializerMap map[string]Initializer

// Register replaces any existing activations.
// It is not concurrent safe, use in init of your package.
func (m initializerMap) Register(name string, fn Initializer) {
	m[name] = fn
}

// Get returns nil if name does not exist.
func (m initializerMap) Get(name string) Initializer {
	return m[name]
}

// binomial returns a []float32 or  []float64 drawn from a binomial distribution given the trial and probability parameters.
func binomial(opts []float64) (InitWFn, error) {
	if len(opts) != 2 {
		return nil, errors.Errorf(parameterFail, "Binomial lnitializer", 2, len(opts))
	}

	trials := opts[0]
	prob := opts[1]

	return func(dt tensor.Dtype, s ...int) interface{} {
		switch dt {
		case tensor.Float64:
			return Binomial64(trials, prob, s...)
		case tensor.Float32:
			return Binomial32(trials, prob, s...)
		default:
			panic(fmt.Sprintf(nyiTypeFail, "Binomial lnitializer", dt))
		}
	}, nil
}

// glorotU creates a InitWFn that populates a Value with weights uniformly sampled using Glorot et al.'s algorithm
func glorotU(opts []float64) (InitWFn, error) {
	if len(opts) != 1 {
		return nil, errors.Errorf(parameterFail, "GlorotU lnitializer", 1, len(opts))
	}
	gain := opts[0]
	return GlorotU(gain), nil
}

// glorotN creates a InitWFn that populates a Value with weights normally sampled using Glorot et al.'s algorithm
func glorotN(opts []float64) (InitWFn, error) {
	if len(opts) != 1 {
		return nil, errors.Errorf(parameterFail, "GlorotN lnitializer", 1, len(opts))
	}
	gain := opts[0]
	return GlorotN(gain), nil
}

// uniform creates a InitWFn with the specified parameters.
// Example Usage:
//		w := NewMatrix(g, Float64, WithName("w"), WithShape(2,2), WithInit(Uniform(-1, 1)))
// This will create a backing slice of []float64, with the length of 4, and its values are drawn from a uniform distro
func uniform(opts []float64) (InitWFn, error) {
	if len(opts) != 2 {
		return nil, errors.Errorf(parameterFail, "Uniform lnitializer", 2, len(opts))
	}
	low := opts[0]
	high := opts[1]
	return Uniform(low, high), nil
}

// gaussian creates a InitWFn with the specified parameters.
// Example Usage:
//		w := NewMatrix(g, Float64, WithName("w"), WithShape(2,2), WithInit(Gaussian(0, 1)))
// This will create a backing slice of []float64, with the length of 4, and its values are drawn from a gaussian distro
func gaussian(opts []float64) (InitWFn, error) {
	if len(opts) != 2 {
		return nil, errors.Errorf(parameterFail, "Gaussian lnitializer", 2, len(opts))
	}
	mean := opts[0]
	stdev := opts[1]
	return Gaussian(mean, stdev), nil
}

// rangedFrom creates an InitWFn that populates a Value starting with the provided start, increamenting the number for each element in the value by 1
func rangedFrom(opts []float64) (InitWFn, error) {
	if len(opts) != 1 {
		return nil, errors.Errorf(parameterFail, "RangedFrom lnitializer", 1, len(opts))
	}
	start := int(opts[0])
	return RangedFrom(start), nil
}

// heEtAlN64 returns float64 weights sampled from a normal distro, using the methods
// described in He et al (2015). The formula is:
//		randn(n) * sqrt(2/n)
// See also https://arxiv.org/abs/1502.01852
//
// For best results, use:
// 		1.0 for gain for weights that will be used in linear and/or sigmoid units
//		math.Sqrt(2.0) for gain for weights that will be used in ReLU units
//		math.Sqrt(2.0 / (1+alpha*alpha)) for ReLU that are leaky with alpha
func heEtAlN(opts []float64) (InitWFn, error) {
	if len(opts) != 1 {
		return nil, errors.Errorf(parameterFail, "HeEtAlN lnitializer", 1, len(opts))
	}
	gain := opts[0]

	return func(dt tensor.Dtype, s ...int) interface{} {
		f64 := HeEtAlN64(gain, s...)
		return F64ToSlice(f64, dt)
	}, nil
}

// heEtAlU64 returns float64 weights sampled from a uniform distro, using the methods
// described in He et al (2015). The formula is:
//		randn(n) * sqrt(2/n)
// See also https://arxiv.org/abs/1502.01852
//
// For best results, use:
// 		1.0 for gain for weights that will be used in linear and/or sigmoid units
//		math.Sqrt(2.0) for gain for weights that will be used in ReLU units
//		math.Sqrt(2.0 / (1+alpha*alpha)) for ReLU that are leaky with alpha
func heEtAlU(opts []float64) (InitWFn, error) {
	if len(opts) != 1 {
		return nil, errors.Errorf(parameterFail, "HeEtAlU lnitializer", 1, len(opts))
	}
	gain := opts[0]

	return func(dt tensor.Dtype, s ...int) interface{} {
		f64 := HeEtAlU64(gain, s...)
		return F64ToSlice(f64, dt)
	}, nil
}

// memset sets all the same value.
func memSet(opts []float64) (InitWFn, error) {
	if len(opts) != 1 {
		return nil, errors.Errorf(parameterFail, "MemSet lnitializer", 1, len(opts))
	}
	v := opts[0]
	return func(dt tensor.Dtype, s ...int) interface{} {
		size := tensor.Shape(s).TotalSize()
		switch dt {
		case tensor.Float64:
			rv := make([]float64, size)
			for i := range rv {
				rv[i] = float64(v)
			}
			return rv
		case tensor.Float32:
			rv := make([]float32, size)
			for i := range rv {
				rv[i] = float32(v)
			}
			return rv
		case tensor.Int:
			rv := make([]int, size)
			for i := range rv {
				rv[i] = int(v)
			}
			return rv
		case tensor.Int32:
			rv := make([]int32, size)
			for i := range rv {
				rv[i] = int32(v)
			}
			return rv
		case tensor.Int64:
			rv := make([]int64, size)
			for i := range rv {
				rv[i] = int64(v)
			}
			return rv
		default:
			panic(fmt.Sprintf(nyiTypeFail, "MemSet lnitializer", dt))
		}
	}, nil
}

func memCopy(opts []float64) (InitWFn, error) {
	if len(opts) == 0 {
		return nil, errors.New("MemSet lnitializer expected 1 parameter at least")
	}
	return func(dt tensor.Dtype, s ...int) interface{} {
		size := tensor.Shape(s).TotalSize()
		if size != len(opts) {
			panic(fmt.Sprintf("shape mismatch, expected total size %v, got %v", size, len(opts)))
		}
		return F64ToSlice(opts, dt)
	}, nil
}

//parseInitStr get function name and float64 options from a string like "Gaussian(0, 0.08)"
func parseInitStr(str string) (name string, opts []float64, err error) {
	strings.TrimSpace(str)
	size := len(str)
	if size == 0 {
		err = ErrorEmptyInitializer
		return
	}
	if str[size-1] != ')' {
		err = errors.Errorf("initializer string error, expected a \")\" at the end of the string, got %s", str[size-1])
		return
	}
	str = str[:size-1]
	strs := strings.Split(str, "(")

	if len(strs) != 2 {
		err = errors.Errorf("initializer string error, expected one \"(\" in the middle of the string, got %v", len(strs))
		return
	}

	//function name
	name = strings.TrimSpace(strs[0])
	optstr := strings.Split(strs[1], ",")

	// parse options as []float64
	var f64 float64
	for _, opt := range optstr {
		f64, err = strconv.ParseFloat(strings.TrimSpace(opt), 64)
		if err != nil {
			err = errors.Errorf("can not parse float from " + opt)
			return
		}
		opts = append(opts, f64)
	}
	return
}
