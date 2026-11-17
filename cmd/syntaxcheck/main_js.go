package main

import "syscall/js"

func main() {
	check := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) != 1 || args[0].Type() != js.TypeString {
			return map[string]any{"error": "invalid syntax-check request"}
		}
		diagnostics, err := checkSource(args[0].String())
		if err != nil {
			return map[string]any{"error": err.Error()}
		}
		values := make([]any, 0, len(diagnostics))
		for _, item := range diagnostics {
			values = append(values, map[string]any{"line": item.Line, "column": item.Column, "message": item.Message})
		}
		return map[string]any{"diagnostics": values}
	})
	js.Global().Set("checkPythonSyntax", check)
	js.Global().Call("syntaxCheckerReady")
	select {}
}
