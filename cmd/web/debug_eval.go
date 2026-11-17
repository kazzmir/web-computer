package main

import (
	"strings"

	"github.com/kazzmir/web-computer/gpython/py"
)

func evaluateDebugExpression(frame *py.Frame, expression string) (py.Object, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" || len(expression) > 10000 {
		return nil, py.ExceptionNewf(py.ValueError, "enter one expression (up to 10000 bytes)")
	}
	code, err := py.Compile(expression, "<debugger expression>", py.EvalMode, 0, true)
	if err != nil {
		return nil, err
	}
	return evaluateDebugCode(frame, code, false)
}

func evaluateDebugCode(frame *py.Frame, code *py.Code, truthValue bool) (py.Object, error) {
	frame.FastToLocals()
	store := frame.Context.Store()
	traceLine, traceFrame := store.TraceLine, store.TraceFrame
	// Calls from the evaluator may execute arbitrary user functions. They must
	// neither hit breakpoints nor become part of the suspended program's stack.
	store.TraceLine, store.TraceFrame = nil, nil
	defer func() {
		store.TraceLine, store.TraceFrame = traceLine, traceFrame
	}()
	value, err := frame.Context.RunCode(code, frame.Globals, frame.Locals, nil)
	if err == nil && truthValue {
		// __bool__/__len__ may themselves execute Python. Keep tracing off
		// until truth conversion is complete too.
		truth, truthErr := py.ObjectIsTrue(value)
		return py.Bool(truth), truthErr
	}
	return value, err
}
