package main

import "github.com/kazzmir/web-computer/gpython/py"

func canStepOut(frame *py.Frame, depth int) bool {
	// Class/module bodies have their own frames too, but are not functions.
	return depth > 1 && frame.Code.Flags&py.CO_NEWLOCALS != 0
}

func shouldPauseDebug(mode string, targetDepth, depth int, breakpoint bool) bool {
	return breakpoint || mode == "into" || mode == "over" && depth <= targetDepth || mode == "out" && depth < targetDepth
}
