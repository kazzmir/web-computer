// Package keyword embeds CPython's keyword lists and membership predicates.
package keyword

import (
	_ "embed"

	"github.com/kazzmir/web-computer/gpython/py"
)

// Copied unchanged from Python-3.15.2/Lib/keyword.py; see test/LICENSE.cpython.
//
//go:embed keyword.py
var source string

func init() {
	py.RegisterModule(&py.ModuleImpl{
		Info:    py.ModuleInfo{Name: "keyword", FileDesc: "keyword.py", Doc: "Python reserved and soft keywords"},
		CodeSrc: source,
	})
}
