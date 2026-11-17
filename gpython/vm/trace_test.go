package vm_test

import (
	"testing"

	"github.com/kazzmir/web-computer/gpython/py"
	"github.com/kazzmir/web-computer/gpython/pytest"
)

func TestLineTrace(t *testing.T) {
	ctx := py.NewContext(py.DefaultContextOpts())
	defer ctx.Close()
	module, code := pytest.CompileSrc(t, ctx, "def add(x):\n    y = x + 1\n    return y\na = 10\nb = add(a)\ni = 0\nwhile i < 3: i += 1\n", "trace.py")
	seenCall, seenFunction, loopHits := false, false, 0
	ctx.Store().TraceLine = func(frame *py.Frame, line int32, depth int) {
		frame.FastToLocals()
		if line == 5 {
			seenCall = true
			if frame.Locals["a"] != py.Int(10) || frame.Locals["b"] != nil {
				t.Error("trace must run before assignment/call")
			}
		}
		if line == 2 {
			seenFunction = true
			if depth != 2 || frame.Locals["x"] != py.Int(10) || frame.Locals["y"] != nil {
				t.Error("function trace must expose arguments before executing its first line")
			}
		}
		if line == 7 {
			loopHits++
		}
	}
	if _, err := ctx.RunCode(code, module.Globals, module.Globals, nil); err != nil {
		t.Fatal(err)
	}
	if !seenCall || !seenFunction || loopHits < 3 {
		t.Fatalf("missing trace events: call=%v function=%v loop=%d", seenCall, seenFunction, loopHits)
	}
	if ctx.Store().RecursionDepth() != 0 {
		t.Fatal("trace leaked frame depth")
	}
}
