package main

import (
	"strings"
	"testing"
)

func TestCheckSource(t *testing.T) {
	for _, source := range []string{"", "x = 3\n", "def f(x):\n    return x\n", "raise AssertionError('must not execute')", "while True:\n    pass", "import missing_module"} {
		diagnostics, err := checkSource(source)
		if err != nil || len(diagnostics) != 0 {
			t.Errorf("valid source %q: %v, %v", source, diagnostics, err)
		}
	}
	for _, test := range []struct {
		source, message string
		line            int
	}{
		{"x = 1\ndef broken(:\n    pass", "SyntaxError", 2},
		{"x = 1\nreturn 3", "outside function", 2},
		{"x = 1\nbreak", "outside loop", 2},
		{"if True:\npass", "", 2},
		{"x = 1\nx = (", "SyntaxError", 2},
	} {
		diagnostics, err := checkSource(test.source)
		if err != nil {
			t.Fatalf("check %q: %v", test.source, err)
		}
		if len(diagnostics) != 1 {
			t.Fatalf("got %v for %q", diagnostics, test.source)
		}
		item := diagnostics[0]
		if item.Line != test.line || item.Column < 1 || !strings.Contains(item.Message, test.message) {
			t.Errorf("got %+v for %q", item, test.source)
		}
	}
}
