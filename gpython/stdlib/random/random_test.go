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
