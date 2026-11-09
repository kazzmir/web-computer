// Package types currently provides dynamic class preparation and creation.
package types

import (
	_ "embed"
	"github.com/kazzmir/web-computer/gpython/py"
)

//go:embed types.py
var source string

func init() {
	py.RegisterModule(&py.ModuleImpl{Info: py.ModuleInfo{Name: "types", Doc: "Dynamic class preparation/creation and runtime alias metadata subset"}, Globals: py.StringDict{"UnionType": py.UnionType, "GenericAlias": py.GenericAliasType}, CodeSrc: source})
}
