// Package io currently supplies the in-memory StringIO text stream subset.
package io

import (
	_ "embed"
	"github.com/kazzmir/web-computer/gpython/py"
)

//go:embed stringio.py
var source string

func init() {
	py.RegisterModule(&py.ModuleImpl{Info: py.ModuleInfo{Name: "io", Doc: "In-memory StringIO text streams (not the full io module)"}, CodeSrc: source})
}
