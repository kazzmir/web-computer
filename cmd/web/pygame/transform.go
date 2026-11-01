package pygame

import (
	"encoding/binary"

	"github.com/kazzmir/web-computer/gpython/py"
)

// Scale2x is the edge-aware pixel-art filter, not nearest-neighbor doubling.
// B, D, F, H are the neighbors above, left, right and below E. At boundaries
// missing neighbors take E's value. Colors (including alpha) are compared and
// copied exactly, without interpolation.
func scale2xRGBA(src []byte, width, height int) []byte {
	dst := make([]byte, width*height*16)
	pixel := func(x, y int) uint32 {
		return binary.LittleEndian.Uint32(src[(y*width+x)*4:])
	}
	put := func(x, y int, c uint32) {
		binary.LittleEndian.PutUint32(dst[(y*width*2+x)*4:], c)
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			e := pixel(x, y)
			b, d, f, h := e, e, e, e
			if y > 0 {
				b = pixel(x, y-1)
			}
			if x > 0 {
				d = pixel(x-1, y)
			}
			if x+1 < width {
				f = pixel(x+1, y)
			}
			if y+1 < height {
				h = pixel(x, y+1)
			}
			tl, tr, bl, br := e, e, e, e
			if b != h && d != f {
				if d == b {
					tl = d
				}
				if b == f {
					tr = f
				}
				if d == h {
					bl = d
				}
				if h == f {
					br = f
				}
			}
			put(2*x, 2*y, tl)
			put(2*x+1, 2*y, tr)
			put(2*x, 2*y+1, bl)
			put(2*x+1, 2*y+1, br)
		}
	}
	return dst
}

func registerTransformMethods(add methodAdder) {
	add("_scale2x", "po", func(args py.Tuple) (py.Object, error) {
		from := args[0].(*pixels)
		size := from.bounds().Size().Mul(2)
		var dest *pixels
		if args[1] != py.None {
			var ok bool
			dest, ok = args[1].(*pixels)
			if !ok {
				return nil, py.ExceptionNewf(py.TypeError, "expected destination surface")
			}
			if dest.bounds().Size() != size {
				return nil, py.ExceptionNewf(py.ValueError, "destination surface must be twice the source size")
			}
		}
		out := newPixels(size.X, size.Y)
		if from.image != nil {
			b := from.bounds()
			data := make([]byte, b.Dx()*b.Dy()*4)
			from.image.ReadPixels(data)
			out.image.WritePixels(scale2xRGBA(data, b.Dx(), b.Dy()))
		}
		if dest == nil {
			return out, nil
		}
		if out.image != nil {
			defer out.image.Deallocate()
		}
		scaleInto(out, dest, false)
		return dest, nil
	})
}
