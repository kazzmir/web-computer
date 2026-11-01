package ctypes_test

import (
	"testing"

	"github.com/kazzmir/web-computer/gpython/py"
	_ "github.com/kazzmir/web-computer/gpython/stdlib"
)

func TestStandaloneModule(t *testing.T) {
	ctx := py.NewContext(py.DefaultContextOpts())
	defer ctx.Close()
	code, err := py.Compile(`
import ctypes
assert ctypes.c_void_p().value is None
assert ctypes.c_void_p(None).value is None
assert ctypes.c_void_p(0).value == 0
assert ctypes.c_void_p(12).value == 12
`, "standalone ctypes test", "exec", 0, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := py.RunCode(ctx, code, "standalone ctypes test", nil); err != nil {
		t.Fatal(err)
	}
}
