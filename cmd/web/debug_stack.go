package main

import (
	"sort"

	"github.com/kazzmir/web-computer/gpython/py"
)

type debugStackEntry struct {
	frame *py.Frame
	line  int32
}
type debugFrameStack struct{ entries []debugStackEntry }

func (stack *debugFrameStack) traceFrame(frame *py.Frame, entering bool) {
	if entering {
		stack.entries = append(stack.entries, debugStackEntry{frame: frame, line: frame.Code.Addr2Line(frame.Lasti)})
	} else {
		last := len(stack.entries) - 1
		stack.entries[last] = debugStackEntry{}
		stack.entries = stack.entries[:last]
	}
}
func (stack *debugFrameStack) traceLine(frame *py.Frame, line int32) {
	if len(stack.entries) > 0 && stack.entries[len(stack.entries)-1].frame == frame {
		stack.entries[len(stack.entries)-1].line = line
	}
}
func (stack *debugFrameStack) frameAt(id int) *py.Frame {
	if id < 0 || id >= len(stack.entries) {
		return nil
	}
	return stack.entries[len(stack.entries)-1-id].frame
}
func (stack *debugFrameStack) snapshot() []any {
	frames := make([]any, 0, len(stack.entries))
	for id := 0; id < len(stack.entries); id++ {
		entry := stack.entries[len(stack.entries)-1-id]
		frames = append(frames, map[string]any{"id": id, "function": entry.frame.Code.Name, "filename": entry.frame.Code.Filename, "line": int(entry.line)})
	}
	return frames
}
func debugLocals(frame *py.Frame) []any {
	frame.FastToLocals()
	names := make([]string, 0, len(frame.Locals))
	for name := range frame.Locals {
		names = append(names, name)
	}
	sort.Strings(names)
	locals := make([]any, 0, len(names))
	for _, name := range names {
		locals = append(locals, map[string]any{"name": name, "value": debugValue(frame.Locals[name], 0)})
	}
	return locals
}
