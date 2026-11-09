package ast

import "github.com/kazzmir/web-computer/gpython/py"

// NamedExpr evaluates Value once and assigns it while retaining the result.
type NamedExpr struct {
	ExprBase
	Target *Name
	Value  Expr
}

var NamedExprType = ExprBaseType.NewType("NamedExpr", "Assignment expression", nil, nil)

func (n *NamedExpr) Type() *py.Type { return NamedExprType }
