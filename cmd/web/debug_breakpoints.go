package main

import (
	"strings"

	"github.com/kazzmir/web-computer/gpython/py"
)

type debugBreakpoint struct {
	condition string
	compiled  bool
	code      *py.Code
	err       error
}

func newDebugBreakpoint(condition string) *debugBreakpoint {
	return &debugBreakpoint{condition: strings.TrimSpace(condition)}
}
func (point *debugBreakpoint) matches(frame *py.Frame) (bool, error) {
	if point.condition == "" {
		return true, nil
	}
	// Lazy compilation runs on the interpreter goroutine, not a JS callback.
	if !point.compiled {
		point.compiled = true
		if len(point.condition) > 10000 {
			point.err = py.ExceptionNewf(py.ValueError, "breakpoint condition exceeds 10000 bytes")
		} else {
			point.code, point.err = py.Compile(point.condition, "<breakpoint condition>", py.EvalMode, 0, true)
		}
	}
	if point.err != nil {
		return false, point.err
	}
	value, err := evaluateDebugCode(frame, point.code, true)
	if err != nil {
		return false, err
	}
	return bool(value.(py.Bool)), nil
}
