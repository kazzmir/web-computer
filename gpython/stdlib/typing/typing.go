// Package typing provides runtime metadata for basic List/Tuple annotations.
package typing

import "github.com/kazzmir/web-computer/gpython/py"

func init() {
	py.RegisterModule(&py.ModuleImpl{
		Info: py.ModuleInfo{Name: "typing", Doc: "Runtime List/Tuple alias metadata and unions (compatibility subset)"},
		Globals: py.StringDict{
			"Union": py.UnionType,
			"List":  &py.TypingAlias{Origin: py.ListType, Name: "typing.List"},
			"Tuple": &py.TypingAlias{Origin: py.TupleType, Name: "typing.Tuple"},
		},
	})
}
