package parser

import (
	"strings"
	"testing"

	"github.com/kazzmir/web-computer/gpython/ast"
	"github.com/kazzmir/web-computer/gpython/py"
)

func TestFStringSyntaxErrors(t *testing.T) {
	for _, source := range []string{
		`f"{"`, `f"}"`, `f"{}"`, `f"{ }"`, `f"{1!x}"`, `f"{1!}"`,
		`f"{1!rr}"`, `f"{1+}"`, `f"{(1]}"`, `f"{[1)}"`, `f"{1:{2:{3:{4}}}}"`,
		`b"bytes" f"{1}"`, `f"{1}" b"bytes"`, `bf"bytes"`, `uf"text"`,
	} {
		t.Run(source, func(t *testing.T) {
			_, err := ParseString(source, py.EvalMode)
			if err == nil {
				t.Fatal("expected SyntaxError")
			}
			if !py.IsException(py.SyntaxError, err) {
				t.Fatalf("expected SyntaxError, got %v", err)
			}
		})
	}
}

func TestFStringAST(t *testing.T) {
	parsed, err := ParseString(`f"value={value!r:>{width}}"`, py.EvalMode)
	if err != nil {
		t.Fatal(err)
	}
	joined, ok := parsed.(*ast.Expression).Body.(*ast.JoinedStr)
	if !ok || len(joined.Values) != 2 {
		t.Fatalf("unexpected AST: %#v", parsed)
	}
	field := joined.Values[1].(*ast.FormattedValue)
	if field.Conversion != 'r' {
		t.Fatalf("conversion = %d", field.Conversion)
	}
	spec := field.FormatSpec.(*ast.JoinedStr)
	if len(spec.Values) != 2 {
		t.Fatalf("format spec = %#v", spec)
	}
}

func TestFStringLineNumbers(t *testing.T) {
	parsed, err := Parse(strings.NewReader("\n\nvalue = f'{missing}'\n"), "example.py", py.ExecMode)
	if err != nil {
		t.Fatal(err)
	}
	ast.Walk(parsed, func(node ast.Ast) bool {
		if name, ok := node.(*ast.Name); ok && name.Id == "missing" && name.GetLineno() != 3 {
			t.Errorf("expression line = %d, want 3", name.GetLineno())
		}
		return true
	})
}
