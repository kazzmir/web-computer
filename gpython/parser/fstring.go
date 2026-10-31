package parser

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/kazzmir/web-computer/gpython/ast"
	"github.com/kazzmir/web-computer/gpython/py"
)

// F-strings remain AST nodes, never calls to eval or to a shadowable builtin.
// Only literal portions undergo escape decoding; expressions are parsed in
// eval mode by the normal parser and participate in symbol-table analysis.
type fstringParser struct {
	source   string
	raw      bool
	pos      ast.Pos
	filename string
	i        int
}

func parseFString(source string, raw bool, pos ast.Pos, filename string) (*ast.JoinedStr, error) {
	p := &fstringParser{source: source, raw: raw, pos: pos, filename: filename}
	return p.parts(false, 0)
}

func (p *fstringParser) parts(spec bool, depth int) (*ast.JoinedStr, error) {
	if depth > 2 {
		return nil, fmt.Errorf("f-string: expressions nested too deeply")
	}
	result := &ast.JoinedStr{ExprBase: ast.ExprBase{Pos: p.pos}}
	var literal strings.Builder
	flush := func() error {
		if literal.Len() == 0 {
			return nil
		}
		text := literal.String()
		if !p.raw {
			decoded, err := DecodeEscape(bytes.NewBufferString(text), false)
			if err != nil {
				return err
			}
			text = decoded.String()
		}
		result.Values = append(result.Values, &ast.Str{ExprBase: ast.ExprBase{Pos: p.pos}, S: py.String(text)})
		literal.Reset()
		return nil
	}
	for p.i < len(p.source) {
		ch := p.source[p.i]
		if ch == '}' && spec {
			break
		}
		if !spec && (ch == '{' || ch == '}') && p.i+1 < len(p.source) && p.source[p.i+1] == ch {
			literal.WriteByte(ch)
			p.i += 2
			continue
		}
		switch ch {
		case '}':
			return nil, fmt.Errorf("f-string: single '}' is not allowed")
		case '{':
			if err := flush(); err != nil {
				return nil, err
			}
			p.i++
			field, debug, err := p.field(depth)
			if err != nil {
				return nil, err
			}
			if debug != "" {
				result.Values = append(result.Values, &ast.Str{ExprBase: ast.ExprBase{Pos: p.pos}, S: py.String(debug)})
			}
			result.Values = append(result.Values, field)
		default:
			// Keep escape sequences intact until decoding. In particular, a brace
			// produced by \u007b is literal text, not a replacement field.
			if ch == '\\' && p.i+1 < len(p.source) && p.source[p.i+1] != '{' && p.source[p.i+1] != '}' {
				literal.WriteByte(ch)
				p.i++
				literal.WriteByte(p.source[p.i])
				p.i++
			} else {
				// A backslash does not escape a field's opening brace. It
				// remains literal even when a flush separates it from '{'.
				if ch == '\\' && !p.raw {
					literal.WriteByte('\\')
				}
				literal.WriteByte(ch)
				p.i++
			}
		}
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *fstringParser) field(depth int) (*ast.FormattedValue, string, error) {
	start := p.i
	var brackets []byte
	quote := byte(0)
	triple := false
	for p.i < len(p.source) {
		ch := p.source[p.i]
		if quote != 0 {
			if ch == '\\' {
				p.i += 2
				continue
			}
			if ch == quote {
				if !triple {
					quote = 0
				} else if strings.HasPrefix(p.source[p.i:], strings.Repeat(string(quote), 3)) {
					quote = 0
					p.i += 2
				}
			}
			p.i++
			continue
		}
		if ch == '\'' || ch == '"' {
			quote = ch
			triple = strings.HasPrefix(p.source[p.i:], strings.Repeat(string(ch), 3))
			if triple {
				p.i += 2
			}
			p.i++
			continue
		}
		if len(brackets) == 0 && (ch == '}' || ch == ':' || ch == '!' && !strings.HasPrefix(p.source[p.i:], "!=") || ch == '=' && !strings.HasPrefix(p.source[p.i:], "==")) {
			break
		}
		switch ch {
		case '(', '[', '{':
			brackets = append(brackets, ch)
		case ')', ']', '}':
			if len(brackets) == 0 {
				return nil, "", fmt.Errorf("f-string: unmatched '%c'", ch)
			}
			open := brackets[len(brackets)-1]
			if open == '(' && ch != ')' || open == '[' && ch != ']' || open == '{' && ch != '}' {
				return nil, "", fmt.Errorf("f-string: mismatched brackets")
			}
			brackets = brackets[:len(brackets)-1]
		case '<', '>', '!', '=':
			// Do not mistake the '=' in a comparison for a debug field marker.
			if p.i+1 < len(p.source) && p.source[p.i+1] == '=' {
				p.i++
			}
		}
		p.i++
	}
	if p.i > len(p.source) {
		return nil, "", fmt.Errorf("f-string: unterminated string in expression")
	}
	expression := strings.TrimSpace(p.source[start:p.i])
	if expression == "" {
		return nil, "", fmt.Errorf("f-string: empty expression not allowed")
	}
	parsed, err := Parse(strings.NewReader("("+expression+")"), p.filename, py.EvalMode)
	if err != nil {
		return nil, "", fmt.Errorf("f-string: invalid expression: %v", err)
	}
	value := parsed.(*ast.Expression).Body
	// Inner parses start at line 1; restore the containing source's line so
	// tracebacks and compiler diagnostics do not point to a synthetic file.
	line := p.pos.Lineno - 1 + strings.Count(p.source[:start], "\n")
	ast.Walk(value, func(node ast.Ast) bool {
		if positioned, ok := node.(interface{ SetPosition(ast.Pos) }); ok {
			positioned.SetPosition(ast.Pos{Lineno: node.GetLineno() + line, ColOffset: node.GetColOffset()})
		}
		return true
	})
	field := &ast.FormattedValue{ExprBase: ast.ExprBase{Pos: p.pos}, Value: value, Conversion: -1}
	debug := ""
	if p.i < len(p.source) && p.source[p.i] == '=' {
		p.i++
		for p.i < len(p.source) && strings.ContainsRune(" \t\r\n", rune(p.source[p.i])) {
			p.i++
		}
		debug = p.source[start:p.i]
	}
	if p.i < len(p.source) && p.source[p.i] == '!' {
		p.i++
		if p.i == len(p.source) || !strings.ContainsRune("sra", rune(p.source[p.i])) {
			return nil, "", fmt.Errorf("f-string: invalid conversion character")
		}
		field.Conversion = int(p.source[p.i])
		p.i++
	}
	if p.i < len(p.source) && p.source[p.i] == ':' {
		p.i++
		field.FormatSpec, err = p.parts(true, depth+1)
		if err != nil {
			return nil, "", err
		}
	} else if debug != "" && field.Conversion == -1 {
		field.Conversion = 'r'
	}
	if p.i == len(p.source) || p.source[p.i] != '}' {
		return nil, "", fmt.Errorf("f-string: expecting '}'")
	}
	p.i++
	return field, debug, nil
}

func concatenateStrings(a, b py.Object) (py.Object, error) {
	if left, ok := a.(py.Bytes); ok {
		if right, ok := b.(py.Bytes); ok {
			return append(left, right...), nil
		}
		return nil, fmt.Errorf("cannot mix bytes and nonbytes literals")
	}
	if _, ok := b.(py.Bytes); ok {
		return nil, fmt.Errorf("cannot mix bytes and nonbytes literals")
	}
	if left, ok := a.(py.String); ok {
		if right, ok := b.(py.String); ok {
			return left + right, nil
		}
	}
	result := &ast.JoinedStr{}
	for _, value := range []py.Object{a, b} {
		if joined, ok := value.(*ast.JoinedStr); ok {
			result.Pos = joined.Pos
			break
		}
	}
	for _, value := range []py.Object{a, b} {
		switch v := value.(type) {
		case py.String:
			result.Values = append(result.Values, &ast.Str{ExprBase: ast.ExprBase{Pos: result.Pos}, S: v})
		case *ast.JoinedStr:
			result.Pos = v.Pos
			result.Values = append(result.Values, v.Values...)
		default:
			return nil, fmt.Errorf("invalid string literal")
		}
	}
	return result, nil
}
