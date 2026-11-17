package main

import (
	"testing"

	"github.com/kazzmir/web-computer/gpython/py"
	"github.com/kazzmir/web-computer/gpython/pytest"
)

func TestEvaluateDebugExpression(t *testing.T) {
	ctx := py.NewContext(py.DefaultContextOpts())
	defer ctx.Close()
	module, code := pytest.CompileSrc(t, ctx, "global_value = 7\ndef twice(n):\n    return n * 2\ndef inner(value, items):\n    local = value + 1\n    result = local\n    return result\ndef outer():\n    value = 20\n    return inner(3, [])\nanswer = outer()", "eval.py")
	stack := &debugFrameStack{}
	ctx.Store().TraceFrame = stack.traceFrame
	evaluated := false
	ctx.Store().TraceLine = func(frame *py.Frame, line int32, depth int) {
		stack.traceLine(frame, line)
		if frame.Code.Name != "inner" || line != 6 {
			return
		}
		evaluated = true
		for _, test := range []struct{ expression, want string }{
			{"local + global_value", "11"}, {"twice(value)", "6"}, {"[value, local]", "[3, 4]"}, {"(value, local)", "(3, 4)"},
			{"{1: local}", "{1: 4}"}, {"items.append(8) or items", "[8]"},
		} {
			result, err := evaluateDebugExpression(frame, test.expression)
			if err != nil || debugValue(result, 0) != test.want {
				t.Errorf("%s: %v, %v", test.expression, result, err)
			}
			if ctx.Store().RecursionDepth() != depth || len(stack.entries) != depth {
				t.Error("evaluation changed suspended stack")
			}
			if ctx.Store().TraceLine == nil || ctx.Store().TraceFrame == nil {
				t.Error("evaluation lost debugger hooks")
			}
		}
		for _, expression := range []string{"", "1 / 0", "missing_name", "value = 9", "twice(1); twice(2)"} {
			if _, err := evaluateDebugExpression(frame, expression); err == nil {
				t.Errorf("expected error for %q", expression)
			}
			if ctx.Store().TraceLine == nil || ctx.Store().TraceFrame == nil {
				t.Error("failed evaluation lost debugger hooks")
			}
		}
		callerResult, err := evaluateDebugExpression(stack.frameAt(1), "value + global_value")
		if err != nil || callerResult != py.Int(27) {
			t.Errorf("caller context: %v, %v", callerResult, err)
		}
		result, err := evaluateDebugExpression(stack.frameAt(2), "global_value")
		if err != nil || result != py.Int(7) {
			t.Errorf("module context: %v, %v", result, err)
		}
	}
	if _, err := ctx.RunCode(code, module.Globals, module.Globals, nil); err != nil {
		t.Fatal(err)
	}
	if !evaluated || module.Globals["answer"] != py.Int(4) || len(stack.entries) != 0 {
		t.Fatal("program did not resume normally after evaluation")
	}
}

func TestEvaluationPreservesClosureSideEffects(t *testing.T) {
	ctx := py.NewContext(py.DefaultContextOpts())
	defer ctx.Close()
	module, code := pytest.CompileSrc(t, ctx, "def outer():\n    value = 1\n    def update():\n        nonlocal value\n        value += 1\n        return value\n    marker = value\n    return value\nanswer = outer()", "closure.py")
	evaluated := false
	ctx.Store().TraceLine = func(frame *py.Frame, line int32, depth int) {
		if frame.Code.Name != "outer" || line != 7 {
			return
		}
		evaluated = true
		result, err := evaluateDebugExpression(frame, "update()")
		if err != nil || result != py.Int(2) {
			t.Fatalf("closure evaluation: %v, %v", result, err)
		}
		frame.FastToLocals()
		if frame.Locals["value"] != py.Int(2) {
			t.Error("stale locals overwrote a closure change")
		}
	}
	if _, err := ctx.RunCode(code, module.Globals, module.Globals, nil); err != nil {
		t.Fatal(err)
	}
	if !evaluated || module.Globals["answer"] != py.Int(2) {
		t.Error("closure side effect did not reach the resumed program")
	}
}
