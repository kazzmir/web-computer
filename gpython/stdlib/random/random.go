// Package random provides a small per-interpreter random-number generator.
package random

import (
	"math/rand"
	"sync"
	"time"

	"github.com/kazzmir/web-computer/gpython/py"
)

type generator struct {
	rng *rand.Rand
	mu  sync.Mutex
}

var randomType = py.NewType("Random", "Random number generator")

func (*generator) Type() *py.Type { return randomType }

func init() {
	randomType.Dict["random"] = py.MustNewMethod("random", func(self py.Object, args py.Tuple) (py.Object, error) {
		if err := py.UnpackTuple(args, nil, "random", 0, 0); err != nil {
			return nil, err
		}
		g := self.(*generator)
		g.mu.Lock()
		defer g.mu.Unlock()
		return py.Float(g.rng.Float64()), nil
	}, 0, "")
	randomType.Dict["seed"] = py.MustNewMethod("seed", func(self py.Object, args py.Tuple) (py.Object, error) {
		var value py.Object = py.None
		if err := py.UnpackTuple(args, nil, "seed", 0, 1, &value); err != nil {
			return nil, err
		}
		seed := time.Now().UnixNano()
		if value != py.None {
			n, ok := value.(py.Int)
			if !ok {
				return nil, py.ExceptionNewf(py.TypeError, "seed must be an integer or None")
			}
			seed = int64(n)
		}
		g := self.(*generator)
		g.mu.Lock()
		defer g.mu.Unlock()
		g.rng.Seed(seed)
		return py.None, nil
	}, 0, "")
	randomType.Dict["_randbelow"] = py.MustNewMethod("_randbelow", func(self py.Object, args py.Tuple) (py.Object, error) {
		var obj py.Object
		if err := py.UnpackTuple(args, nil, "_randbelow", 1, 1, &obj); err != nil {
			return nil, err
		}
		n, ok := obj.(py.Int)
		if !ok || n <= 0 {
			return nil, py.ExceptionNewf(py.ValueError, "random range must be positive")
		}
		g := self.(*generator)
		g.mu.Lock()
		defer g.mu.Unlock()
		return py.Int(g.rng.Int63n(int64(n))), nil
	}, 0, "")
	py.RegisterModule(&py.ModuleImpl{Info: py.ModuleInfo{Name: "random", Doc: "Random numbers, selection, and integer ranges (compatibility subset)"}, Methods: []*py.Method{
		py.MustNewMethod("Random", func(self py.Object, args py.Tuple) (py.Object, error) {
			if err := py.UnpackTuple(args, nil, "Random", 0, 0); err != nil {
				return nil, err
			}
			return &generator{rng: rand.New(rand.NewSource(time.Now().UnixNano()))}, nil
		}, 0, ""),
	}, CodeSrc: `
_inst = Random()
random = _inst.random
seed = _inst.seed

def choice(seq):
    if not len(seq): raise IndexError('cannot choose from an empty sequence')
    return seq[_inst._randbelow(len(seq))]

def randrange(start, stop=None, step=1):
    if stop is None: start, stop = 0, start
    if step == 0: raise ValueError('zero step for randrange')
    count = len(range(start, stop, step))
    return start + step * _inst._randbelow(count)

def randint(a, b): return randrange(a, b+1)
`})
}
