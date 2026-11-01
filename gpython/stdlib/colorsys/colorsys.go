// Package colorsys embeds CPython's RGB/HSV/HLS/YIQ conversion functions.
package colorsys

import (
	_ "embed"

	"github.com/kazzmir/web-computer/gpython/py"
)

// Copied unchanged from Python-3.15.2/Lib/colorsys.py; see test/LICENSE.cpython.
//
//go:embed colorsys.py
var source string

func init() {
	py.RegisterModule(&py.ModuleImpl{
		Info:    py.ModuleInfo{Name: "colorsys", FileDesc: "colorsys.py", Doc: "Color-system conversions"},
		CodeSrc: source,
	})
}
