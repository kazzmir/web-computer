package pygame

import (
	_ "embed"
	"encoding/binary"
	"math"

	"github.com/kazzmir/web-computer/gpython/py"
)

//go:embed numpy.py
var numpySource string

func arrayBytes(obj py.Object) ([]byte, error) {
	dtype, e := py.GetAttrString(obj, "dtype")
	if e != nil {
		return nil, e
	}
	data, e := py.GetAttrString(obj, "_data")
	if e != nil {
		return nil, e
	}
	fields, e := py.GetAttrString(dtype, "fields")
	if e != nil {
		return nil, e
	}
	var out []byte
	appendNumber := func(value, typ py.Object) error {
		name, e := py.GetAttrString(typ, "name")
		if e != nil {
			return e
		}
		n, e := py.FloatAsFloat64(value)
		if e != nil {
			return e
		}
		switch name {
		case py.String("float32"):
			out = binary.LittleEndian.AppendUint32(out, math.Float32bits(float32(n)))
		case py.String("float64"):
			out = binary.LittleEndian.AppendUint64(out, math.Float64bits(n))
		case py.String("uint32"):
			out = binary.LittleEndian.AppendUint32(out, uint32(int64(n)))
		default:
			return py.ExceptionNewf(py.TypeError, "unsupported array dtype")
		}
		return nil
	}
	if fields == py.None {
		values, e := py.SequenceTuple(data)
		if e != nil {
			return nil, e
		}
		for _, v := range values {
			if e := appendNumber(v, dtype); e != nil {
				return nil, e
			}
		}
	} else {
		descriptors, e := py.SequenceTuple(fields)
		if e != nil {
			return nil, e
		}
		shape, e := py.GetAttrString(obj, "shape")
		if e != nil {
			return nil, e
		}
		dimensions, e := py.SequenceTuple(shape)
		if e != nil {
			return nil, e
		}
		if len(dimensions) != 1 {
			return nil, py.ExceptionNewf(py.ValueError, "structured buffers must be one dimensional")
		}
		n, ok := dimensions[0].(py.Int)
		if !ok {
			return nil, py.ExceptionNewf(py.TypeError, "invalid buffer shape")
		}
		for row := 0; row < int(n); row++ {
			for _, desc := range descriptors {
				field, e := py.SequenceTuple(desc)
				if e != nil {
					return nil, e
				}
				if len(field) != 3 {
					return nil, py.ExceptionNewf(py.TypeError, "invalid field descriptor")
				}
				width, ok := field[2].(py.Int)
				if !ok || width < 1 {
					return nil, py.ExceptionNewf(py.ValueError, "invalid field width")
				}
				values, e := py.GetItem(data, field[0])
				if e != nil {
					return nil, e
				}
				for col := 0; col < int(width); col++ {
					value, e := py.GetItem(values, py.Int(row*int(width)+col))
					if e != nil {
						return nil, e
					}
					if e := appendNumber(value, field[1]); e != nil {
						return nil, e
					}
				}
			}
		}
	}
	return out, nil
}

func registerGLDependencies() {
	py.RegisterModule(&py.ModuleImpl{Info: py.ModuleInfo{Name: "numpy", Doc: "Matrix and structured-buffer subset for the GL emulator"}, CodeSrc: numpySource, Methods: []*py.Method{
		py.MustNewMethod("_array_bytes", func(self py.Object, args py.Tuple) (py.Object, error) {
			var obj py.Object
			if e := py.UnpackTuple(args, nil, "tobytes", 1, 1, &obj); e != nil {
				return nil, e
			}
			bytes, e := arrayBytes(obj)
			if e != nil {
				return nil, e
			}
			return py.Bytes(bytes), nil
		}, 0, ""),
	}})
	py.RegisterModule(&py.ModuleImpl{Info: py.ModuleInfo{Name: "ctypes", Doc: "Safe GL buffer offsets only; no native FFI"}, CodeSrc: "class c_void_p:\n    def __init__(self, value=None): self.value = value\n"})
}
