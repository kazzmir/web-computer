package sys_test

import (
	"github.com/kazzmir/web-computer/gpython/py"
	_ "github.com/kazzmir/web-computer/gpython/stdlib"
	"testing"
)

func TestRecursionLimitsAreContextLocal(t *testing.T) {
	first := py.NewContext(py.DefaultContextOpts())
	defer first.Close()
	second := py.NewContext(py.DefaultContextOpts())
	defer second.Close()
	for _, test := range []struct {
		ctx    py.Context
		source string
	}{
		{first, `import sys
assert sys.getrecursionlimit() == 1000
sys.setrecursionlimit(64)
assert sys.getrecursionlimit() == 64
`},
		{second, `import sys
assert sys.getrecursionlimit() == 1000
sys.setrecursionlimit(90)
`},
		{first, `import sys
assert sys.getrecursionlimit() == 64
def recurse(): return recurse()
try: recurse()
except RecursionError: pass
else: assert False
assert sys.getrecursionlimit() == 64
`},
	} {
		code, err := py.Compile(test.source, "recursion regression", py.ExecMode, 0, false)
		if err == nil {
			_, err = py.RunCode(test.ctx, code, "recursion regression", nil)
		}
		if err != nil {
			py.TracebackDump(err)
			t.Fatal(err)
		}
		if depth := test.ctx.Store().RecursionDepth(); depth != 0 {
			t.Fatalf("execution depth leaked: %d", depth)
		}
	}
}
