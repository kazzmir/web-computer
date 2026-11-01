package random_test

import (
	"testing"

	"github.com/kazzmir/web-computer/gpython/py"
	_ "github.com/kazzmir/web-computer/gpython/stdlib"
)

func TestRandom(t *testing.T) {
	ctx := py.NewContext(py.DefaultContextOpts())
	defer ctx.Close()
	code, err := py.Compile(`
import random
random.seed(17)
a = [random.random() for i in range(10)]
random.seed(17)
assert a == [random.random() for i in range(10)]
assert all([0 <= x < 1 for x in a])
assert random.choice(['only']) == 'only'
assert all([random.choice((-1, 1)) in (-1, 1) for i in range(50)])
assert all([random.randrange(1, 8, 2) in (1, 3, 5, 7) for i in range(50)])
assert all([3 <= random.randint(3, 5) <= 5 for i in range(50)])
start, stop = 1 << 1000, 1 << 1001
random.seed(17)
large = [random.randrange(start, stop, 7) for i in range(10)]
assert all([start <= x < stop and (x-start) % 7 == 0 for x in large])
random.seed(17)
assert large == [random.randrange(start, stop, 7) for i in range(10)]
assert all([start < random.randrange(stop, start, -7) <= stop for i in range(10)])
for call in (lambda: random.randrange(0), lambda: random.randrange(3, 1), lambda: random.randrange(1, 3, -1), lambda: random.randrange(1, 3, 0)):
    try: call()
    except ValueError: pass
    else: assert False
for call in (lambda: random.randrange(1.2), lambda: random.randrange(1, 3.5), lambda: random.randrange(1, 3, 1.0)):
    try: call()
    except TypeError: pass
    else: assert False
try:
    random.choice([])
except IndexError:
    pass
else:
    assert False
`, "random test", "exec", 0, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = py.RunCode(ctx, code, "random test", nil); err != nil {
		t.Fatal(err)
	}
}
