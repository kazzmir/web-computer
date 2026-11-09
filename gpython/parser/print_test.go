package parser

import (
	"github.com/kazzmir/web-computer/gpython/py"
	"strings"
	"testing"
)

func TestPrintMigrationHints(t *testing.T) {
	for _, source := range []string{`print "hello"`, `print value;`, `for x in xs: print x`} {
		_, err := ParseString(source, py.ExecMode)
		if err == nil || !strings.Contains(err.Error(), "Missing parentheses in call to 'print'") {
			t.Errorf("%q: %v", source, err)
		}
	}
	for _, source := range []string{`print("hello")`, `print = 1`, `text = "print hello"`, `def f(print): return print`} {
		if _, err := ParseString(source, py.ExecMode); err != nil {
			t.Errorf("%q: %v", source, err)
		}
	}
	for _, source := range []string{`text = "print hello"; ]`, `# print hello
x = ]`, `print + ]`, `obj.print value`, `f(print value)`} {
		_, err := ParseString(source, py.ExecMode)
		if err == nil || strings.Contains(err.Error(), "Missing parentheses") {
			t.Errorf("unrelated syntax error %q: %v", source, err)
		}
	}
}
