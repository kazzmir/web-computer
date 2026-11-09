// Package testsupport supplies the explicitly limited helpers needed by the
// imported CPython regression tests. Helpers perform real capture/restoration
// and garbage collection; this module does not skip tests.
package testsupport

import (
	_ "embed"
	"github.com/kazzmir/web-computer/gpython/py"
	"runtime"
)

//go:embed support.py
var source string

func init() {
	py.RegisterModule(&py.ModuleImpl{
		Info: py.ModuleInfo{Name: "test.support", Doc: "Limited regression helpers: captured_stdout, swap_attr, gc_collect"},
		Methods: []*py.Method{py.MustNewMethod("gc_collect", func(self py.Object) (py.Object, error) {
			runtime.GC()
			runtime.Gosched()
			runtime.GC()
			return py.None, nil
		}, 0, "Run Go's garbage collector and allow pending finalizers to run.")},
		CodeSrc: source,
	})
}
