package main

import (
	"fmt"
	"sort"
	"syscall/js"

	"github.com/kazzmir/web-computer/gpython/py"
)

// The command callback only wakes the Python goroutine; it never runs Python
// inside a JavaScript callback (which would block the browser event loop).
func setupDebugger(ctx py.Context, filename string, engine *Engine) func() {
	config := js.Global().Get("webComputerDebug")
	if config.Type() != js.TypeObject || !config.Get("enabled").Bool() {
		return func() {}
	}
	breakpoints := map[int32]bool{}
	update := func(lines js.Value) {
		clear(breakpoints)
		for i := 0; i < lines.Length(); i++ {
			breakpoints[int32(lines.Index(i).Int())] = true
		}
	}
	update(config.Get("breakpoints"))
	commands := make(chan string, 1)
	command := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		if args[0].String() == "breakpoints" && len(args) > 1 {
			update(args[1])
			return nil
		}
		select {
		case commands <- args[0].String():
		default:
		}
		return nil
	})
	js.Global().Set("webComputerDebugCommand", command)
	mode, targetDepth := "continue", 0
	ctx.Store().TraceLine = func(frame *py.Frame, line int32, depth int) {
		// Only user source is debuggable; imported library internals are skipped.
		if frame.Code.Filename != filename {
			return
		}
		if !breakpoints[line] && mode != "into" && !(mode == "over" && depth <= targetDepth) {
			return
		}
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
		js.Global().Call("webComputerPaused", map[string]any{"line": int(line), "function": frame.Code.Name, "locals": locals})
		// Keep Ebitengine drawing the last presented frame while Python is
		// paused. Blocking Update on the command channel can leave a cleared
		// browser framebuffer visible until execution resumes.
		for {
			select {
			case mode = <-commands:
				targetDepth = depth
				return
			default:
				if err := engine.Yield(); err != nil {
					return
				}
			}
		}
	}
	return func() {
		ctx.Store().TraceLine = nil
		js.Global().Set("webComputerDebugCommand", js.Undefined())
		command.Release()
	}
}

// Avoid calling user-defined __repr__ while inspecting a paused interpreter.
func debugValue(value py.Object, depth int) string {
	if value == nil {
		return "<unbound>"
	}
	switch v := value.(type) {
	case py.String:
		text := string(v)
		if len(text) > 300 {
			text = text[:300] + "…"
		}
		return fmt.Sprintf("%q", text)
	case py.Int, py.Float, py.Bool, py.NoneType:
		text, _ := py.ReprAsString(value)
		return text
	case *py.List:
		if depth >= 2 {
			return "[…]"
		}
		text := "["
		for i, item := range v.Items {
			if i > 0 {
				text += ", "
			}
			if i == 10 {
				text += "…"
				break
			}
			text += debugValue(item, depth+1)
		}
		return text + "]"
	default:
		return "<" + string(value.Type().Name) + ">"
	}
}
