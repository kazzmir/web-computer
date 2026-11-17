package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kazzmir/web-computer/gpython/ast"
	"github.com/kazzmir/web-computer/gpython/parser"
	"github.com/kazzmir/web-computer/gpython/py"
)

// Parse literals only: editing a value must not execute calls or other Python
// expressions in the paused interpreter. A bare identifier is a string too.
func parseDebugValue(text string) (py.Object, error) {
	text = strings.TrimSpace(text)
	if len(text) == 0 || len(text) > 10000 {
		return nil, fmt.Errorf("enter a value (up to 10000 characters)")
	}
	tree, err := parser.ParseString(text, py.EvalMode)
	if err != nil {
		return nil, fmt.Errorf("invalid value: use a quoted string, number, list, or dictionary")
	}
	expr := tree.(*ast.Expression).Body
	if name, ok := expr.(*ast.Name); ok {
		return py.String(name.Id), nil
	}
	return debugLiteral(expr, 0)
}

func debugLiteral(expr ast.Expr, depth int) (py.Object, error) {
	if depth > 30 {
		return nil, fmt.Errorf("value is nested too deeply")
	}
	switch node := expr.(type) {
	case *ast.Str:
		return node.S, nil
	case *ast.Num:
		return py.Object(node.N), nil
	case *ast.NameConstant:
		return py.Object(node.Value), nil
	case *ast.UnaryOp:
		if _, ok := node.Operand.(*ast.Num); ok {
			value, err := debugLiteral(node.Operand, depth+1)
			if err != nil {
				return nil, err
			}
			if node.Op == ast.USub {
				return py.Neg(value)
			}
			if node.Op == ast.UAdd {
				return py.Pos(value)
			}
		}
	case *ast.List:
		items := make([]py.Object, len(node.Elts))
		for i, expr := range node.Elts {
			value, err := debugLiteral(expr, depth+1)
			if err != nil {
				return nil, err
			}
			items[i] = value
		}
		return py.NewListFromItems(items), nil
	case *ast.Dict:
		var result py.Object = py.NewStringDict()
		for i, expr := range node.Keys {
			key, err := debugLiteral(expr, depth+1)
			if err != nil {
				return nil, err
			}
			value, err := debugLiteral(node.Values[i], depth+1)
			if err != nil {
				return nil, err
			}
			result, err = py.DictBuilderSet(result, key, value)
			if err != nil {
				return nil, fmt.Errorf("invalid dictionary key")
			}
		}
		return result, nil
	}
	return nil, fmt.Errorf("use literal values only: strings, numbers, lists, dictionaries, True, False, or None")
}

func setDebugVariable(frame *py.Frame, name, text string) (py.Object, error) {
	frame.FastToLocals()
	if _, exists := frame.Locals[name]; !exists {
		return nil, fmt.Errorf("variable is no longer available")
	}
	value, err := parseDebugValue(text)
	if err != nil {
		return nil, err
	}
	frame.Locals[name] = value
	// Function arguments, optimized locals, and closure cells must be updated,
	// not just their inspection dictionary. Module locals share their globals.
	frame.LocalsToFast(false)
	return value, nil
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
	case py.Tuple:
		if depth >= 2 {
			return "(…)"
		}
		parts := []string{}
		for i, item := range v {
			if i == 10 {
				parts = append(parts, "…")
				break
			}
			parts = append(parts, debugValue(item, depth+1))
		}
		suffix := ""
		if len(v) == 1 {
			suffix = ","
		}
		return "(" + strings.Join(parts, ", ") + suffix + ")"
	case *py.List:
		if depth >= 2 {
			return "[…]"
		}
		parts := []string{}
		for i, item := range v.Items {
			if i == 10 {
				parts = append(parts, "…")
				break
			}
			parts = append(parts, debugValue(item, depth+1))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case py.StringDict:
		if depth >= 2 {
			return "{…}"
		}
		names := make([]string, 0, len(v))
		for name := range v {
			names = append(names, name)
		}
		sort.Strings(names)
		parts := []string{}
		for i, name := range names {
			if i == 10 {
				parts = append(parts, "…")
				break
			}
			parts = append(parts, debugValue(py.String(name), depth+1)+": "+debugValue(v[name], depth+1))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case *py.GenericDict:
		if depth >= 2 {
			return "{…}"
		}
		parts := []string{}
		for i, pair := range v.SnapshotItems() {
			if i == 10 {
				parts = append(parts, "…")
				break
			}
			parts = append(parts, debugValue(pair[0], depth+1)+": "+debugValue(pair[1], depth+1))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	default:
		return "<" + value.Type().Name + ">"
	}
}
