package main

import (
	"strings"
	"testing"

	"github.com/kazzmir/web-computer/gpython/py"
	"github.com/kazzmir/web-computer/gpython/pytest"
)

func TestParseDebugValue(t *testing.T) {
	for _, test := range []struct{ input, want string }{
		{`"sdf"`, `'sdf'`}, {`sdf`, `'sdf'`}, {`3`, `3`}, {`8.2`, `8.2`},
		{`-3`, `-3`}, {`+8.2`, `8.2`}, {`[1, 2, 3]`, `[1, 2, 3]`},
		{`{1: "a"}`, `{1: 'a'}`}, {`{"a": [1, -2.5, None]}`, `{'a': [1, -2.5, None]}`},
		{`True`, `True`}, {`False`, `False`}, {`None`, `None`},
	} {
		t.Run(test.input, func(t *testing.T) {
			value, err := parseDebugValue(test.input)
			if err != nil {
				t.Fatal(err)
			}
			repr, err := py.ReprAsString(value)
			if err != nil || repr != test.want {
				t.Fatalf("got %q, %v; want %q", repr, err, test.want)
			}
		})
	}
	for _, input := range []string{"", "[1,", "print('unsafe')", "1 + 2", "[x for x in []]", "{[]: 1}", "[unknown]", strings.Repeat("[", 32) + "1" + strings.Repeat("]", 32)} {
		if _, err := parseDebugValue(input); err == nil {
			t.Errorf("accepted non-literal or invalid input %q", input)
		}
	}
}

func TestSetDebugVariableUpdatesFastLocalsAndCells(t *testing.T) {
	ctx := py.NewContext(py.DefaultContextOpts())
	defer ctx.Close()
	module, code := pytest.CompileSrc(t, ctx, "def outer(x):\n    value = x\n    def inner():\n        return value\n    return inner()\nanswer = outer(1)\n", "edit.py")
	editedArgument, editedCell := false, false
	ctx.Store().TraceLine = func(frame *py.Frame, line int32, depth int) {
		if frame.Code.Name != "outer" {
			return
		}
		name, text := "", ""
		if line == 2 {
			name, text = "x", "8.2"
			editedArgument = true
		}
		if line == 5 {
			frame.FastToLocals()
			if frame.Locals["value"] != py.Float(8.2) {
				t.Error("edited argument not used")
			}
			name, text = "value", "42"
			editedCell = true
		}
		if name == "" {
			return
		}
		if _, err := setDebugVariable(frame, name, text); err != nil {
			t.Error(err)
		}
		before := frame.Locals[name]
		if _, err := setDebugVariable(frame, name, "bad()"); err == nil {
			t.Error("accepted a call")
		}
		if frame.Locals[name] != before {
			t.Error("invalid edit changed variable")
		}
		if _, err := setDebugVariable(frame, "missing", "3"); err == nil {
			t.Error("created nonexistent variable")
		}
	}
	if _, err := ctx.RunCode(code, module.Globals, module.Globals, nil); err != nil {
		t.Fatal(err)
	}
	if !editedArgument || !editedCell || module.Globals["answer"] != py.Int(42) {
		t.Fatalf("edits did not reach execution: %v", module.Globals["answer"])
	}
}

func TestDebugValueDictionaryPreview(t *testing.T) {
	value, err := parseDebugValue(`{1: "a"}`)
	if err != nil {
		t.Fatal(err)
	}
	if got := debugValue(value, 0); got != `{1: "a"}` {
		t.Fatalf("preview = %q", got)
	}
	// Recursive containers must remain bounded.
	list := py.NewList()
	list.Items = append(list.Items, list)
	if got := debugValue(list, 0); got != `[[[…]]]` {
		t.Fatalf("recursive preview = %q", got)
	}
}
