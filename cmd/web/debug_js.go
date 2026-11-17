package main

import (
	"syscall/js"

	"github.com/kazzmir/web-computer/gpython/py"
)

type debugCommand struct {
	action, name, value string
	frameID             int
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
		cmd := debugCommand{action: action, frameID: -1}
		if (action == "set-variable" || action == "select-frame") && len(args) > 2 && args[2].Type() == js.TypeObject {
			if id := args[2].Get("frameId"); id.Type() == js.TypeNumber {
				cmd.frameID = id.Int()
			}
			if action == "set-variable" {
				cmd.name = args[2].Get("name").String()
				cmd.value = args[2].Get("value").String()
			}
		}
		select {
		case commands <- cmd:
		default:
		}
		return nil
	})
	js.Global().Set("webComputerDebugCommand", command)
	mode, targetDepth := "continue", 0
	stack := &debugFrameStack{}
	ctx.Store().TraceFrame = stack.traceFrame
	ctx.Store().TraceLine = func(frame *py.Frame, line int32, depth int) {
		stack.traceLine(frame, line)
		if frame.Code.Filename != filename {
			return
		}
		if !shouldPauseDebug(mode, targetDepth, depth, breakpoints[line]) {
			return
		}
		selectedID := 0
		selectedFrame := frame
		paused = true
		defer func() { paused = false }()
		js.Global().Call("webComputerPaused", map[string]any{"line": int(line), "function": frame.Code.Name, "locals": debugLocals(frame), "frames": stack.snapshot(), "canStepOut": canStepOut(frame, depth)})
		// Yield to Ebitengine even while paused so the last frame stays visible.
		for {
			select {
			case cmd := <-commands:
				if cmd.action == "select-frame" {
					if candidate := stack.frameAt(cmd.frameID); candidate != nil {
						selectedID, selectedFrame = cmd.frameID, candidate
						js.Global().Call("webComputerFrameSelected", map[string]any{"frameId": selectedID, "locals": debugLocals(selectedFrame)})
					} else {
						js.Global().Call("webComputerFrameSelected", map[string]any{"frameId": cmd.frameID, "error": "Frame is no longer available"})
					}
					continue
				}
				if cmd.action == "set-variable" {
					result := map[string]any{"name": cmd.name, "frameId": cmd.frameID}
					if cmd.frameID != selectedID {
						result["error"] = "Selected frame changed"
						js.Global().Call("webComputerVariableResult", result)
						continue
					}
					value, err := setDebugVariable(selectedFrame, cmd.name, cmd.value)
					if err != nil {
						result["error"] = err.Error()
					} else {
						result["value"] = debugValue(value, 0)
					}
					js.Global().Call("webComputerVariableResult", result)
					continue
				}
				if cmd.action == "out" && !canStepOut(frame, depth) {
					continue
				}
				if cmd.action == "continue" || cmd.action == "into" || cmd.action == "over" || cmd.action == "out" {
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
		ctx.Store().TraceFrame = nil
		js.Global().Set("webComputerDebugCommand", js.Undefined())
		command.Release()
	}
}
