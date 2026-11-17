package main

import (
	"sort"
	"syscall/js"

	"github.com/kazzmir/web-computer/gpython/py"
)

type debugCommand struct {
	action, name, value string
}

// The callback only queues commands; Python values are changed in the paused
// interpreter goroutine, never in a browser event callback.
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
	commands := make(chan debugCommand, 1)
	paused := false
	command := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		action := args[0].String()
		if action == "breakpoints" && len(args) > 1 {
			update(args[1])
			return nil
		}
		if !paused {
			return nil
		}
		cmd := debugCommand{action: action}
		if action == "set-variable" && len(args) > 2 {
			cmd.name = args[2].Get("name").String()
			cmd.value = args[2].Get("value").String()
		}
		select {
		case commands <- cmd:
		default:
		}
		return nil
	})
	js.Global().Set("webComputerDebugCommand", command)
	mode, targetDepth := "continue", 0
	ctx.Store().TraceLine = func(frame *py.Frame, line int32, depth int) {
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
		paused = true
		defer func() { paused = false }()
		js.Global().Call("webComputerPaused", map[string]any{"line": int(line), "function": frame.Code.Name, "locals": locals})
		// Yield to Ebitengine even while paused so the last frame stays visible.
		for {
			select {
			case cmd := <-commands:
				if cmd.action == "set-variable" {
					value, err := setDebugVariable(frame, cmd.name, cmd.value)
					result := map[string]any{"name": cmd.name}
					if err != nil {
						result["error"] = err.Error()
					} else {
						result["value"] = debugValue(value, 0)
					}
					js.Global().Call("webComputerVariableResult", result)
					continue
				}
				if cmd.action == "continue" || cmd.action == "into" || cmd.action == "over" {
					mode, targetDepth = cmd.action, depth
					return
				}
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
