package main

import (
	"testing"

	"github.com/kazzmir/web-computer/gpython/py"
	"github.com/kazzmir/web-computer/gpython/pytest"
)

func TestConditionalBreakpoints(t *testing.T) {
	ctx := py.NewContext(py.DefaultContextOpts())
	defer ctx.Close()
	module, code := pytest.CompileSrc(t, ctx, "class Flag:\n    def __bool__(self):\n        return True\ndef predicate(n):\n    return n == 3\ni = 0\nwhile i < 5:\n    i += 1", "condition.py")
	point := newDebugBreakpoint("i == 3")
	hits := []int{}
	var compiled *py.Code
	ctx.Store().TraceLine = func(frame *py.Frame, line int32, depth int) {
		if frame.Code.Name == "predicate" || frame.Code.Name == "__bool__" || frame.Code.Filename == "<breakpoint condition>" {
			t.Error("condition calls must not recursively trace")
		}
		if frame.Code.Name != "<module>" || line != 8 {
			return
		}
		hit, err := point.matches(frame)
		if err != nil {
			t.Fatal(err)
		}
		if hit {
			hits = append(hits, int(frame.Locals["i"].(py.Int)))
		}
		if compiled == nil {
			compiled = point.code
		} else if compiled != point.code {
			t.Error("condition recompiled on every hit")
		}
		for _, condition := range []string{"predicate(3)", "Flag()", "[1]"} {
			if ok, err := newDebugBreakpoint(condition).matches(frame); !ok || err != nil {
				t.Errorf("truth conversion/call failed: %s: %v", condition, err)
			}
		}
		if ok, err := newDebugBreakpoint("False").matches(frame); ok || err != nil {
			t.Error("false condition should not stop")
		}
		for _, condition := range []string{"missing_name > 3", "i = 3", "1 / 0"} {
			if _, err := newDebugBreakpoint(condition).matches(frame); err == nil {
				t.Errorf("expected condition error for %q", condition)
			}
		}
		if ctx.Store().TraceLine == nil || ctx.Store().RecursionDepth() != depth {
			t.Error("condition evaluation changed tracing/frame state")
		}
	}
	if _, err := ctx.RunCode(code, module.Globals, module.Globals, nil); err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0] != 3 || module.Globals["i"] != py.Int(5) {
		t.Fatalf("wrong conditional stops: %v", hits)
	}
}
