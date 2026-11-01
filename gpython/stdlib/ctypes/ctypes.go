// Package ctypes provides pointer-value records without exposing native memory
// or FFI. This is a deliberately limited subset of Python's ctypes module.
package ctypes

import "github.com/kazzmir/web-computer/gpython/py"

func init() {
	py.RegisterModule(&py.ModuleImpl{
		Info:    py.ModuleInfo{Name: "ctypes", Doc: "Safe pointer-value records; no native memory or FFI"},
		CodeSrc: "class c_void_p:\n    def __init__(self, value=None): self.value = value\n",
	})
}
