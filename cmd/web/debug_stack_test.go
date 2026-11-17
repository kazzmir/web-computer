package main

import (
	"testing"

	"github.com/kazzmir/web-computer/gpython/py"
	"github.com/kazzmir/web-computer/gpython/pytest"
)

func TestDebugStackFramesAndCallerLocals(t *testing.T) {
	ctx := py.NewContext(py.DefaultContextOpts())
	defer ctx.Close()
	module, code := pytest.CompileSrc(t, ctx, "def inner(x):\n    local = x + 1\n    return local\ndef outer():\n    saved = 10\n    result = inner(saved)\n    return result + saved\nanswer = outer()", "stack.py")
	stack := &debugFrameStack{}
	ctx.Store().TraceFrame = stack.traceFrame
	inspected := false
	ctx.Store().TraceLine = func(frame *py.Frame, line int32, depth int) {
		stack.traceLine(frame, line)
		if frame.Code.Name != "inner" || line != 2 {
			return
		}
		inspected = true
		snapshot := stack.snapshot()
		for id, expected := range []struct {
			name string
			line int
		}{{"inner", 2}, {"outer", 6}, {"<module>", 8}} {
			item := snapshot[id].(map[string]any)
			if item["id"] != id || item["function"] != expected.name || item["line"] != expected.line {
				t.Errorf("frame %d: %v", id, item)
			}
		}
		caller := stack.frameAt(1)
		caller.FastToLocals()
		if caller.Locals["saved"] != py.Int(10) {
			t.Error("missing caller locals")
		}
		if _, err := setDebugVariable(caller, "saved", "50"); err != nil {
			t.Fatal(err)
		}
		if stack.frameAt(-1) != nil || stack.frameAt(3) != nil {
			t.Error("invalid frame ids must be rejected")
		}
	}
	if _, err := ctx.RunCode(code, module.Globals, module.Globals, nil); err != nil {
		t.Fatal(err)
	}
	if !inspected || module.Globals["answer"] != py.Int(61) {
		t.Fatalf("caller edit did not affect execution: %v", module.Globals["answer"])
	}
	if len(stack.entries) != 0 {
		t.Fatal("stack entries leaked on return")
	}
}

func TestDebugStackUnwindsAndGeneratorResumes(t *testing.T) {
	for i, source := range []string{
		"def f():\n    raise ValueError('bad')\nf()",
		"def gen():\n    yield 1\n    yield 2\ng = gen()\na = next(g)\nb = next(g)",
		"def f(n):\n    if n:\n        return f(n - 1)\n    return n\nf(3)",
	} {
		ctx := py.NewContext(py.DefaultContextOpts())
		module, code := pytest.CompileSrc(t, ctx, source, "unwind.py")
		stack := &debugFrameStack{}
		ctx.Store().TraceFrame = stack.traceFrame
		ctx.Store().TraceLine = func(frame *py.Frame, line int32, depth int) {
			stack.traceLine(frame, line)
			if len(stack.entries) != depth || stack.frameAt(0) != frame {
				t.Error("active stack disagrees with VM depth")
			}
		}
		_, err := ctx.RunCode(code, module.Globals, module.Globals, nil)
		if (i == 0) != (err != nil) {
			t.Errorf("unexpected execution result: %v", err)
		}
		if len(stack.entries) != 0 {
			t.Errorf("stack did not unwind for %q", source)
		}
		ctx.Close()
	}
}
