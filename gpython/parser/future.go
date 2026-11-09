package parser

import (
	"github.com/kazzmir/web-computer/gpython/ast"
	"github.com/kazzmir/web-computer/gpython/py"
)

type comparisonToken struct {
	spelling string
	pos      ast.Pos
	line     string
}

func FutureFlags(module ast.Mod, flags int) int {
	var statements []ast.Stmt
	switch module := module.(type) {
	case *ast.Module:
		statements = module.Body
	case *ast.Interactive:
		statements = module.Body
	}
	// Only the actual module preamble can enable a future: neither strings nor
	// relative imports nor imports nested in a function change parser behavior.
	for i, statement := range statements {
		if expression, ok := statement.(*ast.ExprStmt); ok && i == 0 {
			if _, docstring := expression.Value.(*ast.Str); docstring {
				continue
			}
		}
		directive, ok := statement.(*ast.ImportFrom)
		if !ok || directive.Level != 0 || directive.Module != "__future__" {
			break
		}
		for _, name := range directive.Names {
			if name.Name == "barry_as_FLUFL" {
				flags |= py.CO_FUTURE_BARRY_AS_BDFL
			}
		}
	}
	return flags
}

func (lex *yyLex) checkComparisons(flags int) error {
	barry := FutureFlags(lex.mod, flags)&py.CO_FUTURE_BARRY_AS_BDFL != 0
	for _, token := range lex.comparisons {
		message := ""
		if barry && token.spelling == "!=" {
			message = "with Barry as BDFL, use '<>' instead of '!='"
		}
		if !barry && token.spelling == "<>" {
			message = "invalid syntax"
		}
		if message != "" {
			return py.MakeSyntaxError(py.ExceptionNewf(py.SyntaxError, "%s", message), lex.filename, token.pos.Lineno, token.pos.ColOffset, token.line)
		}
	}
	return nil
}
