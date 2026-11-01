package numpy_test

import (
	"encoding/binary"
	"math"
	"testing"

	_ "github.com/kazzmir/web-computer/gpython/lib/numpy"
	"github.com/kazzmir/web-computer/gpython/py"
	_ "github.com/kazzmir/web-computer/gpython/stdlib"
)

// This package deliberately imports no pygame, OpenGL, or Ebitengine code.
func TestStandaloneModule(t *testing.T) {
	ctx := py.NewContext(py.DefaultContextOpts())
	defer ctx.Close()
	code, err := py.Compile(`
import numpy as np
m = np.eye(3, dtype=np.float32)
assert m.shape == (3, 3)
assert m.strides == (12, 4)
assert m.nbytes == 36
m[0, 2] = 5
assert m.T[2, 0] == 5
assert np.dot(np.eye(3), m).tolist() == [[1, 0, 5], [0, 1, 0], [0, 0, 1]]
m[...] = np.zeros((3, 3), dtype=m.dtype)
assert m.tolist() == [[0, 0, 0], [0, 0, 0], [0, 0, 0]]
values = np.array([1, 2, 3], dtype=np.uint32)
assert len(values) == 3 and values.nbytes == 12
unsigned_bytes = values.tobytes()
float_bytes = np.array([1.5, -2.0], dtype=np.float32).tobytes()
vertices = np.zeros(2, [('position', np.float32, 3), ('color', np.float32, 4)])
vertices['position'] = [[1, 2, 3], [4, 5, 6]]
vertices['color'] = [[1, 0, 0, 1], [0, 1, 0, 1]]
assert vertices.nbytes == 56 and vertices.strides == (28,)
assert vertices.dtype['position'].itemsize == 12
structured_bytes = vertices.tobytes()
`, "standalone numpy test", "exec", 0, false)
	if err != nil {
		t.Fatal(err)
	}
	mod, err := py.RunCode(ctx, code, "standalone numpy test", nil)
	if err != nil {
		t.Fatal(err)
	}
	uints := mod.Globals["unsigned_bytes"].(py.Bytes)
	if len(uints) != 12 {
		t.Fatalf("uint32 buffer length = %d, want 12", len(uints))
	}
	for i, want := range []uint32{1, 2, 3} {
		if got := binary.LittleEndian.Uint32(uints[i*4:]); got != want {
			t.Errorf("uint32 element %d = %d, want %d", i, got, want)
		}
	}
	floats := mod.Globals["float_bytes"].(py.Bytes)
	if len(floats) != 8 {
		t.Fatalf("float32 buffer length = %d, want 8", len(floats))
	}
	for i, want := range []float32{1.5, -2} {
		if got := math.Float32frombits(binary.LittleEndian.Uint32(floats[i*4:])); got != want {
			t.Errorf("float32 element %d = %g, want %g", i, got, want)
		}
	}
	structured := mod.Globals["structured_bytes"].(py.Bytes)
	if len(structured) != 56 {
		t.Fatalf("structured buffer length = %d, want 56", len(structured))
	}
	for i, want := range []float32{1, 2, 3, 1, 0, 0, 1, 4, 5, 6, 0, 1, 0, 1} {
		if got := math.Float32frombits(binary.LittleEndian.Uint32(structured[i*4:])); got != want {
			t.Errorf("structured element %d = %g, want %g", i, got, want)
		}
	}
}
