package main

import (
	"syscall/js"

	"github.com/kazzmir/web-computer/gpython/py"
)

type debugCommand struct {
	action, name, value, expression string
	frameID, requestID              int
}

// Send through the existing iframe message protocol instead of JS callback
// globals. A cached runner must not crash a newer debugger on frame selection.
func sendDebuggerMessage(kind string, fields map[string]any) {
	fields["channel"] = "web-computer"
	fields["type"] = kind
	js.Global().Get("parent").Call("postMessage", fields, js.Global().Get("location").Get("origin"))
}

// The callback only queues commands; Python values are changed in the paused
// interpreter goroutine, never in a browser event callback.
func setupDebugger(ctx py.Context, filename string, engine *Engine) func() {
	config := js.Global().Get("webComputerDebug")
	if config.Type() != js.TypeObject || !config.Get("enabled").Bool() {
		return func() {}
	}
	breakpoints := map[int32]*debugBreakpoint{}
	update := func(lines js.Value) {
		next := map[int32]*debugBreakpoint{}
		for i := 0; i < lines.Length(); i++ {
			item := lines.Index(i)
			line, condition := 0, ""
			if item.Type() == js.TypeNumber {
				line = item.Int()
			} else if item.Type() == js.TypeObject && !item.IsNull() && item.Get("line").Type() == js.TypeNumber {
				line = item.Get("line").Int()
				if value := item.Get("condition"); value.Type() == js.TypeString {
					condition = value.String()
				}
			}
			if line < 1 {
				continue
			}
			point := newDebugBreakpoint(condition)
			if old := breakpoints[int32(line)]; old != nil && old.condition == point.condition {
				point = old
			}
			next[int32(line)] = point
		}
		breakpoints = next
	}
	update(config.Get("breakpoints"))
	commands := make(chan debugCommand, 1)
	paused := false
	breakRequested := false
	command := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		action := args[0].String()
		if action == "breakpoints" && len(args) > 1 {
			update(args[1])
			return nil
		}
		if action == "break" {
			if !paused {
				breakRequested = true
			}
			return nil
		}
		if !paused {
			return nil
		}
		cmd := debugCommand{action: action, frameID: -1, requestID: -1}
		if (action == "set-variable" || action == "select-frame" || action == "evaluate") && len(args) > 2 && args[2].Type() == js.TypeObject {
			if id := args[2].Get("frameId"); id.Type() == js.TypeNumber {
				cmd.frameID = id.Int()
			}
			if action == "evaluate" {
				cmd.expression = args[2].Get("expression").String()
				if id := args[2].Get("requestId"); id.Type() == js.TypeNumber {
					cmd.requestID = id.Int()
				}
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
		breakpointHit := false
		var conditionError error
		if !breakRequested && frame.Code.Filename == filename {
			if point := breakpoints[line]; point != nil {
				breakpointHit, conditionError = point.matches(frame)
				if conditionError != nil {
					breakpointHit = true
				}
			}
		}
		if !breakRequested && (frame.Code.Filename != filename || !shouldPauseDebug(mode, targetDepth, depth, breakpointHit)) {
			return
		}
		// A manual break may stop in library code too. Consume the request at
		// any pause so Continue cannot accidentally re-trigger it.
		breakRequested = false
		selectedID := 0
		selectedFrame := frame
		paused = true
		defer func() { paused = false }()
		state := map[string]any{"line": int(line), "filename": frame.Code.Filename, "function": frame.Code.Name, "locals": debugLocals(frame), "frames": stack.snapshot(), "canStepOut": canStepOut(frame, depth)}
		if conditionError != nil {
			state["conditionError"] = conditionError.Error()
		}
		sendDebuggerMessage("paused", state)
		// Yield to Ebitengine even while paused so the last frame stays visible.
		for {
			select {
			case cmd := <-commands:
				if cmd.action == "evaluate" {
					result := map[string]any{"frameId": cmd.frameID, "requestId": cmd.requestID}
					if cmd.frameID != selectedID {
						result["error"] = "Selected frame changed"
					} else {
						value, err := evaluateDebugExpression(selectedFrame, cmd.expression)
						if err != nil {
							result["error"] = err.Error()
						} else {
							result["value"] = debugValue(value, 0)
						}
					}
					result["locals"] = debugLocals(selectedFrame)
					sendDebuggerMessage("evaluation-result", result)
					continue
				}
				if cmd.action == "select-frame" {
					if candidate := stack.frameAt(cmd.frameID); candidate != nil {
						selectedID, selectedFrame = cmd.frameID, candidate
						sendDebuggerMessage("frame-selected", map[string]any{"frameId": selectedID, "locals": debugLocals(selectedFrame)})
					} else {
						sendDebuggerMessage("frame-selected", map[string]any{"frameId": cmd.frameID, "error": "Frame is no longer available"})
					}
					continue
				}
				if cmd.action == "set-variable" {
					result := map[string]any{"name": cmd.name, "frameId": cmd.frameID}
					if cmd.frameID != selectedID {
						result["error"] = "Selected frame changed"
						sendDebuggerMessage("variable-result", result)
						continue
					}
					value, err := setDebugVariable(selectedFrame, cmd.name, cmd.value)
					if err != nil {
						result["error"] = err.Error()
					} else {
						result["value"] = debugValue(value, 0)
					}
					sendDebuggerMessage("variable-result", result)
					continue
				}
				if cmd.action == "out" && !canStepOut(frame, depth) {
					continue
				}
				if cmd.action == "continue" || cmd.action == "into" || cmd.action == "over" || cmd.action == "out" {
					mode, targetDepth = cmd.action, depth
					sendDebuggerMessage("debug-resumed", map[string]any{})
					return
				}
			default:
				if err := engine.Yield(); err != nil {
					return
				}
			}
		}
	}
	sendDebuggerMessage("debug-ready", map[string]any{"canEvaluate": true})
	return func() {
		ctx.Store().TraceLine = nil
		ctx.Store().TraceFrame = nil
		js.Global().Set("webComputerDebugCommand", js.Undefined())
		command.Release()
	}
}
