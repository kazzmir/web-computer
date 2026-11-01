package pygame

import (
	"encoding/binary"
	"math"
	"regexp"
	"strings"

	"github.com/kazzmir/web-computer/gpython/py"
)

const (
	glLines             = 1
	glTriangles         = 4
	glQuads             = 7
	glDepthTest         = 0x0B71
	glBlend             = 0x0BE2
	glPolygonOffsetFill = 0x8037
	glLineSmooth        = 0x0B20
	glLess              = 0x0201
	glLequal            = 0x0203
	glAlways            = 0x0207
	glProjection        = 0x1701
	glModelview         = 0x1700
	glColorBit          = 0x4000
	glDepthBit          = 0x0100
	glVertexShader      = 0x8B31
	glFragmentShader    = 0x8B30
	glCompileStatus     = 0x8B81
	glLinkStatus        = 0x8B82
	glValidateStatus    = 0x8B83
	glInfoLogLength     = 0x8B84
	glArrayBuffer       = 0x8892
	glElementBuffer     = 0x8893
	glFloat             = 0x1406
	glUnsignedInt       = 0x1405
	glUnsignedShort     = 0x1403
	glUnsignedByte      = 0x1401
)

type glShader struct {
	stage       int
	source, log string
	compiled    bool
}
type glProgram struct {
	attached                                      map[int]*glShader
	linked, validated, transforms, colorModifiers bool
	log                                           string
	attrs                                         map[string]int
	locations                                     map[string]int
	types                                         map[string]string
	matrices                                      map[string]glMatrix
	vectors                                       map[string]glVector
}
type glAttribute struct {
	buffer, size, kind, stride, offset int
	normalized, enabled                bool
}
type glVAO struct {
	attrs   map[int]glAttribute
	element int
}
type glUniform struct {
	program int
	name    string
}

func glInt(obj py.Object) (int, error) {
	if v, ok := obj.(py.Int); ok {
		return int(v), nil
	}
	return 0, py.ExceptionNewf(py.TypeError, "expected integer GL argument")
}
func glInts(args py.Tuple) ([]int, error) {
	out := make([]int, len(args))
	for i, v := range args {
		n, e := glInt(v)
		if e != nil {
			return nil, e
		}
		out[i] = n
	}
	return out, nil
}
func glNumbers(args py.Tuple) ([]float64, error) {
	out := make([]float64, len(args))
	for i, v := range args {
		n, e := py.FloatAsFloat64(v)
		if e != nil {
			return nil, e
		}
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return nil, py.ExceptionNewf(py.ValueError, "GL argument must be finite")
		}
		out[i] = n
	}
	return out, nil
}
func glBoolean(v py.Object) (bool, error) {
	if b, ok := v.(py.Bool); ok {
		return bool(b), nil
	}
	i, e := glInt(v)
	return i != 0, e
}
func glOffset(v py.Object) (int, error) {
	if v == py.None {
		return 0, nil
	}
	if _, ok := v.(py.Int); !ok {
		var e error
		v, e = py.GetAttrString(v, "value")
		if e != nil {
			return 0, e
		}
		if v == py.None {
			return 0, nil
		}
	}
	n, e := glInt(v)
	if e != nil {
		return 0, e
	}
	if n < 0 {
		return 0, py.ExceptionNewf(py.ValueError, "negative buffer offset")
	}
	return n, nil
}
func glFlat(v py.Object) ([]float64, error) {
	if data, e := py.GetAttrString(v, "_data"); e == nil {
		v = data
	}
	var out []float64
	var visit func(py.Object) error
	visit = func(value py.Object) error {
		switch value.(type) {
		case py.Int, py.Float, py.Bool, *py.BigInt:
			n, e := py.FloatAsFloat64(value)
			if e != nil {
				return e
			}
			out = append(out, n)
			return nil
		}
		switch value.(type) {
		case py.String, py.Bytes:
			return py.ExceptionNewf(py.TypeError, "expected numeric GL data, not a string")
		}
		seq, e := py.SequenceTuple(value)
		if e != nil {
			return e
		}
		for _, x := range seq {
			if e := visit(x); e != nil {
				return e
			}
		}
		return nil
	}
	if e := visit(v); e != nil {
		return nil, e
	}
	return out, nil
}
func (g *glContext) id() int { n := g.nextID; g.nextID++; return n }
func (g *glContext) shaderObject(v py.Object) (*glShader, error) {
	id, e := glInt(v)
	if e != nil {
		return nil, e
	}
	s, ok := g.shaders[id]
	if !ok {
		return nil, py.ExceptionNewf(py.ValueError, "unknown GL shader %d", id)
	}
	return s, nil
}
func (g *glContext) programObject(v py.Object) (*glProgram, error) {
	id, e := glInt(v)
	if e != nil {
		return nil, e
	}
	p, ok := g.programs[id]
	if !ok {
		return nil, py.ExceptionNewf(py.ValueError, "unknown GL program %d", id)
	}
	return p, nil
}
func (g *glContext) uniform(v py.Object) (*glProgram, string, error) {
	id, e := glInt(v)
	if e != nil {
		return nil, "", e
	}
	if id == -1 {
		return nil, "", nil
	}
	u, ok := g.uniforms[id]
	if !ok || u.program != g.program {
		return nil, "", py.ExceptionNewf(py.ValueError, "uniform does not belong to the active program")
	}
	return g.programs[u.program], u.name, nil
}

var glUniformPattern = regexp.MustCompile(`uniform\s+(mat4|vec4)\s+(\w+)\s*;`)
var glCommentPattern = regexp.MustCompile(`(?s)/\*.*?\*/|//[^\n]*`)

// This is an explicit compatibility shader contract, not a GLSL compiler.
// Reject extra shader operations instead of silently rendering them incorrectly.
func shaderBody(source string) string {
	text := strings.Join(strings.Fields(glCommentPattern.ReplaceAllString(source, "")), "")
	start := strings.Index(text, "voidmain(){")
	if start < 0 || !strings.HasSuffix(text, "}") {
		return ""
	}
	return text[start+len("voidmain(){") : len(text)-1]
}

func shaderContract(s *glShader) bool {
	body := shaderBody(s.source)
	if s.stage == glFragmentShader {
		return body == "fragColor=vertex_color_out;"
	}
	statements := strings.Split(body, ";")
	if len(statements) != 3 || statements[2] != "" {
		return false
	}
	position, color := false, false
	for _, statement := range statements[:2] {
		switch statement {
		case "gl_Position=projection*view*model*vec4(vertex_position,1.0)", "gl_Position=vec4(vertex_position,1.0)":
			position = true
		case "vertex_color_out=(colour_mul*vertex_colour)+colour_add", "vertex_color_out=vertex_colour":
			color = true
		default:
			return false
		}
	}
	return position && color
}

func (g *glContext) registerModules() {
	globals := py.StringDict{}
	for name, value := range map[string]int{
		"GL_LINES": glLines, "GL_TRIANGLES": glTriangles, "GL_QUADS": glQuads, "GL_DEPTH_TEST": glDepthTest, "GL_BLEND": glBlend, "GL_POLYGON_OFFSET_FILL": glPolygonOffsetFill, "GL_LINE_SMOOTH": glLineSmooth,
		"GL_LESS": glLess, "GL_LEQUAL": glLequal, "GL_ALWAYS": glAlways, "GL_PROJECTION": glProjection, "GL_MODELVIEW": glModelview, "GL_COLOR_BUFFER_BIT": glColorBit, "GL_DEPTH_BUFFER_BIT": glDepthBit,
		"GL_VERTEX_SHADER": glVertexShader, "GL_FRAGMENT_SHADER": glFragmentShader, "GL_ARRAY_BUFFER": glArrayBuffer, "GL_ELEMENT_ARRAY_BUFFER": glElementBuffer, "GL_FLOAT": glFloat, "GL_UNSIGNED_INT": glUnsignedInt, "GL_UNSIGNED_SHORT": glUnsignedShort, "GL_UNSIGNED_BYTE": glUnsignedByte,
		"GL_COMPILE_STATUS": glCompileStatus, "GL_LINK_STATUS": glLinkStatus, "GL_VALIDATE_STATUS": glValidateStatus, "GL_INFO_LOG_LENGTH": glInfoLogLength,
		"GL_DYNAMIC_DRAW": 0x88E8, "GL_STATIC_DRAW": 0x88E4, "GL_SRC_ALPHA": 0x0302, "GL_ONE_MINUS_SRC_ALPHA": 0x0303, "GL_ONE": 1, "GL_ZERO": 0, "GL_FALSE": 0, "GL_TRUE": 1, "GL_NO_ERROR": 0,
		"GL_LINE_SMOOTH_HINT": 0x0C52, "GL_NICEST": 0x1102, "GL_VERSION": 0x1F02, "GL_RENDERER": 0x1F01, "GL_VENDOR": 0x1F00,
	} {
		globals[name] = py.Int(value)
	}
	var methods []*py.Method
	add := func(name string, count int, fn nativeFunc) {
		methods = append(methods, py.MustNewMethod(name, func(self py.Object, a py.Tuple) (py.Object, error) {
			if len(a) != count {
				return nil, py.ExceptionNewf(py.TypeError, "%s expects %d arguments", name, count)
			}
			return fn(a)
		}, 0, ""))
	}
	ints := func(name string, count int, fn func([]int) error) {
		add(name, count, func(a py.Tuple) (py.Object, error) {
			v, e := glInts(a)
			if e != nil {
				return nil, e
			}
			return py.None, fn(v)
		})
	}
	numbers := func(name string, count int, fn func([]float64) error) {
		add(name, count, func(a py.Tuple) (py.Object, error) {
			v, e := glNumbers(a)
			if e != nil {
				return nil, e
			}
			return py.None, fn(v)
		})
	}
	add("glCreateShader", 1, func(a py.Tuple) (py.Object, error) {
		stage, e := glInt(a[0])
		if e != nil {
			return nil, e
		}
		if stage != glVertexShader && stage != glFragmentShader {
			return nil, py.ExceptionNewf(py.ValueError, "unsupported shader stage")
		}
		id := g.id()
		g.shaders[id] = &glShader{stage: stage}
		return py.Int(id), nil
	})
	add("glShaderSource", 2, func(a py.Tuple) (py.Object, error) {
		s, e := g.shaderObject(a[0])
		if e != nil {
			return nil, e
		}
		source, ok := a[1].(py.String)
		if !ok {
			return nil, py.ExceptionNewf(py.TypeError, "shader source must be a string")
		}
		s.source = string(source)
		s.compiled = false
		return py.None, nil
	})
	add("glCompileShader", 1, func(a py.Tuple) (py.Object, error) {
		s, e := g.shaderObject(a[0])
		if e != nil {
			return nil, e
		}
		s.compiled = shaderContract(s)
		s.log = ""
		if !s.compiled {
			s.log = "Ebitengine GL emulator supports transform/vertex-color shaders only (see README)."
		}
		return py.None, nil
	})
	add("glGetShaderInfoLog", 1, func(a py.Tuple) (py.Object, error) {
		s, e := g.shaderObject(a[0])
		if e != nil {
			return nil, e
		}
		return py.String(s.log), nil
	})
	add("glGetShaderiv", 2, func(a py.Tuple) (py.Object, error) {
		s, e := g.shaderObject(a[0])
		if e != nil {
			return nil, e
		}
		kind, e := glInt(a[1])
		if e != nil {
			return nil, e
		}
		if kind == glInfoLogLength {
			return py.Int(len(s.log)), nil
		}
		if kind != glCompileStatus {
			return nil, py.ExceptionNewf(py.ValueError, "unsupported shader query")
		}
		if s.compiled {
			return py.Int(1), nil
		}
		return py.Int(0), nil
	})
	add("glCreateProgram", 0, func(a py.Tuple) (py.Object, error) {
		id := g.id()
		g.programs[id] = &glProgram{attached: map[int]*glShader{}, attrs: map[string]int{}, locations: map[string]int{}, types: map[string]string{}, matrices: map[string]glMatrix{}, vectors: map[string]glVector{}}
		return py.Int(id), nil
	})
	add("glAttachShader", 2, func(a py.Tuple) (py.Object, error) {
		p, e := g.programObject(a[0])
		if e != nil {
			return nil, e
		}
		s, e := g.shaderObject(a[1])
		if e != nil {
			return nil, e
		}
		id, _ := glInt(a[1])
		p.attached[id] = s
		return py.None, nil
	})
	add("glDetachShader", 2, func(a py.Tuple) (py.Object, error) {
		p, e := g.programObject(a[0])
		if e != nil {
			return nil, e
		}
		id, e := glInt(a[1])
		if e != nil {
			return nil, e
		}
		delete(p.attached, id)
		return py.None, nil
	})
	add("glLinkProgram", 1, func(a py.Tuple) (py.Object, error) {
		p, e := g.programObject(a[0])
		if e != nil {
			return nil, e
		}
		var vs, fs *glShader
		for _, s := range p.attached {
			if s.stage == glVertexShader {
				vs = s
			} else {
				fs = s
			}
		}
		p.linked = vs != nil && fs != nil && vs.compiled && fs.compiled
		p.log = ""
		if !p.linked {
			p.log = "A compiled supported vertex and fragment shader are required."
			return py.None, nil
		}
		p.attrs = map[string]int{"vertex_position": 0, "vertex_colour": 1}
		for _, match := range glUniformPattern.FindAllStringSubmatch(vs.source+"\n"+fs.source, -1) {
			p.types[match[2]] = match[1]
		}
		p.transforms = strings.Contains(shaderBody(vs.source), "projection*view*model")
		p.colorModifiers = strings.Contains(shaderBody(vs.source), "colour_mul")
		return py.None, nil
	})
	add("glValidateProgram", 1, func(a py.Tuple) (py.Object, error) {
		p, e := g.programObject(a[0])
		if e != nil {
			return nil, e
		}
		p.validated = p.linked
		return py.None, nil
	})
	add("glGetProgramInfoLog", 1, func(a py.Tuple) (py.Object, error) {
		p, e := g.programObject(a[0])
		if e != nil {
			return nil, e
		}
		return py.String(p.log), nil
	})
	add("glGetProgramiv", 2, func(a py.Tuple) (py.Object, error) {
		p, e := g.programObject(a[0])
		if e != nil {
			return nil, e
		}
		kind, e := glInt(a[1])
		if e != nil {
			return nil, e
		}
		if kind == glInfoLogLength {
			return py.Int(len(p.log)), nil
		}
		status := false
		switch kind {
		case glLinkStatus:
			status = p.linked
		case glValidateStatus:
			status = p.validated
		default:
			return nil, py.ExceptionNewf(py.ValueError, "unsupported program query")
		}
		if status {
			return py.Int(1), nil
		}
		return py.Int(0), nil
	})
	ints("glUseProgram", 1, func(v []int) error {
		if v[0] != 0 {
			p, ok := g.programs[v[0]]
			if !ok || !p.linked {
				return py.ExceptionNewf(py.ValueError, "program is not linked")
			}
		}
		g.program = v[0]
		return nil
	})
	add("glGetAttribLocation", 2, func(a py.Tuple) (py.Object, error) {
		p, e := g.programObject(a[0])
		if e != nil {
			return nil, e
		}
		name, ok := a[1].(py.String)
		if !ok {
			return nil, py.ExceptionNewf(py.TypeError, "attribute name must be a string")
		}
		if loc, ok := p.attrs[string(name)]; ok {
			return py.Int(loc), nil
		}
		return py.Int(-1), nil
	})
	add("glGetUniformLocation", 2, func(a py.Tuple) (py.Object, error) {
		p, e := g.programObject(a[0])
		if e != nil {
			return nil, e
		}
		name, ok := a[1].(py.String)
		if !ok {
			return nil, py.ExceptionNewf(py.TypeError, "uniform name must be a string")
		}
		n := string(name)
		if _, ok := p.types[n]; !ok {
			return py.Int(-1), nil
		}
		if id, ok := p.locations[n]; ok {
			return py.Int(id), nil
		}
		id := g.id()
		program, _ := glInt(a[0])
		g.uniforms[id] = glUniform{program, n}
		p.locations[n] = id
		return py.Int(id), nil
	})
	add("glUniform4f", 5, func(a py.Tuple) (py.Object, error) {
		p, name, e := g.uniform(a[0])
		if e != nil {
			return nil, e
		}
		v, e := glNumbers(a[1:])
		if e != nil {
			return nil, e
		}
		if p != nil {
			if p.types[name] != "vec4" {
				return nil, py.ExceptionNewf(py.TypeError, "uniform is not vec4")
			}
			p.vectors[name] = glVector{v[0], v[1], v[2], v[3]}
		}
		return py.None, nil
	})
	add("glUniformMatrix4fv", 4, func(a py.Tuple) (py.Object, error) {
		p, name, e := g.uniform(a[0])
		if e != nil {
			return nil, e
		}
		count, e := glInt(a[1])
		if e != nil {
			return nil, e
		}
		transpose, e := glBoolean(a[2])
		if e != nil {
			return nil, e
		}
		values, e := glFlat(a[3])
		if e != nil {
			return nil, e
		}
		if count != 1 || len(values) != 16 {
			return nil, py.ExceptionNewf(py.ValueError, "expected one 4x4 matrix")
		}
		if p != nil {
			if p.types[name] != "mat4" {
				return nil, py.ExceptionNewf(py.TypeError, "uniform is not mat4")
			}
			var matrix glMatrix
			copy(matrix[:], values)
			if transpose {
				for col := 0; col < 4; col++ {
					for row := 0; row < 4; row++ {
						matrix[col*4+row] = values[row*4+col]
					}
				}
			}
			p.matrices[name] = matrix
		}
		return py.None, nil
	})
	for _, definition := range []struct {
		name string
		vao  bool
	}{{"glGenBuffers", false}, {"glGenVertexArrays", true}} {
		name, vao := definition.name, definition.vao
		add(name, 1, func(a py.Tuple) (py.Object, error) {
			n, e := glInt(a[0])
			if e != nil {
				return nil, e
			}
			if n < 0 || n > 65536 {
				return nil, py.ExceptionNewf(py.ValueError, "invalid GL object count")
			}
			ids := make(py.Tuple, n)
			for i := range ids {
				id := g.id()
				ids[i] = py.Int(id)
				if vao {
					g.vaos[id] = &glVAO{attrs: map[int]glAttribute{}}
				} else {
					g.buffers[id] = nil
				}
			}
			if n == 1 {
				return ids[0], nil
			}
			return ids, nil
		})
	}
	ints("glBindVertexArray", 1, func(v []int) error {
		if _, ok := g.vaos[v[0]]; !ok {
			return py.ExceptionNewf(py.ValueError, "unknown vertex array")
		}
		g.vao = v[0]
		return nil
	})
	ints("glBindBuffer", 2, func(v []int) error {
		if v[1] != 0 {
			if _, ok := g.buffers[v[1]]; !ok {
				return py.ExceptionNewf(py.ValueError, "unknown buffer")
			}
		}
		switch v[0] {
		case glArrayBuffer:
			g.arrayBuffer = v[1]
		case glElementBuffer:
			g.vaos[g.vao].element = v[1]
		default:
			return py.ExceptionNewf(py.ValueError, "unsupported buffer target")
		}
		return nil
	})
	add("glBufferData", 4, func(a py.Tuple) (py.Object, error) {
		target, e := glInt(a[0])
		if e != nil {
			return nil, e
		}
		size, e := glInt(a[1])
		if e != nil {
			return nil, e
		}
		if size < 0 || size > 64*1024*1024 {
			return nil, py.ExceptionNewf(py.ValueError, "invalid buffer size")
		}
		usage, e := glInt(a[3])
		if e != nil {
			return nil, e
		}
		if usage != 0x88E8 && usage != 0x88E4 {
			return nil, py.ExceptionNewf(py.ValueError, "unsupported buffer usage hint")
		}
		id := 0
		switch target {
		case glArrayBuffer:
			id = g.arrayBuffer
		case glElementBuffer:
			id = g.vaos[g.vao].element
		default:
			return nil, py.ExceptionNewf(py.ValueError, "unsupported buffer target")
		}
		if id == 0 {
			return nil, py.ExceptionNewf(py.ValueError, "no buffer bound")
		}
		data := make([]byte, size)
		if a[2] != py.None {
			value := a[2]
			if _, ok := value.(py.Bytes); !ok {
				method, e := py.GetAttrString(value, "tobytes")
				if e != nil {
					return nil, e
				}
				value, e = py.Call(method, nil, nil)
				if e != nil {
					return nil, e
				}
			}
			raw, ok := value.(py.Bytes)
			if !ok {
				return nil, py.ExceptionNewf(py.TypeError, "buffer data must provide bytes")
			}
			if len(raw) < size {
				return nil, py.ExceptionNewf(py.ValueError, "buffer data is shorter than size")
			}
			copy(data, raw[:size])
		}
		g.buffers[id] = data
		return py.None, nil
	})
	for _, enabled := range []bool{true, false} {
		name := "glDisableVertexAttribArray"
		if enabled {
			name = "glEnableVertexAttribArray"
		}
		enabled := enabled
		ints(name, 1, func(v []int) error {
			if v[0] < 0 {
				return py.ExceptionNewf(py.ValueError, "negative attribute location")
			}
			a := g.vaos[g.vao].attrs[v[0]]
			a.enabled = enabled
			g.vaos[g.vao].attrs[v[0]] = a
			return nil
		})
	}
	add("glVertexAttribPointer", 6, func(a py.Tuple) (py.Object, error) {
		v, e := glInts(a[:3])
		if e != nil {
			return nil, e
		}
		normalized, e := glBoolean(a[3])
		if e != nil {
			return nil, e
		}
		stride, e := glInt(a[4])
		if e != nil {
			return nil, e
		}
		offset, e := glOffset(a[5])
		if e != nil {
			return nil, e
		}
		if g.arrayBuffer == 0 || v[0] < 0 || v[1] < 1 || v[1] > 4 || stride < 0 || (v[2] != glFloat && v[2] != glUnsignedByte) {
			return nil, py.ExceptionNewf(py.ValueError, "invalid vertex attribute format or no buffer bound")
		}
		old := g.vaos[g.vao].attrs[v[0]]
		g.vaos[g.vao].attrs[v[0]] = glAttribute{g.arrayBuffer, v[1], v[2], stride, offset, normalized, old.enabled}
		return py.None, nil
	})
	add("glDrawElements", 4, func(a py.Tuple) (py.Object, error) {
		v, e := glInts(a[:3])
		if e != nil {
			return nil, e
		}
		offset, e := glOffset(a[3])
		if e != nil {
			return nil, e
		}
		size := 0
		switch v[2] {
		case glUnsignedInt:
			size = 4
		case glUnsignedShort:
			size = 2
		case glUnsignedByte:
			size = 1
		default:
			return nil, py.ExceptionNewf(py.ValueError, "unsupported element type")
		}
		data := g.buffers[g.vaos[g.vao].element]
		if v[1] < 0 || offset > len(data) || v[1] > (len(data)-offset)/size {
			return nil, py.ExceptionNewf(py.ValueError, "index range outside buffer")
		}
		indices := make([]int, v[1])
		for i := range indices {
			b := data[offset+i*size:]
			switch size {
			case 4:
				indices[i] = int(binary.LittleEndian.Uint32(b))
			case 2:
				indices[i] = int(binary.LittleEndian.Uint16(b))
			case 1:
				indices[i] = int(b[0])
			}
		}
		return py.None, g.drawIndexed(v[0], indices)
	})
	ints("glDrawArrays", 3, func(v []int) error {
		if v[1] < 0 || v[2] < 0 || v[2] > 1000000 {
			return py.ExceptionNewf(py.ValueError, "invalid vertex range")
		}
		indices := make([]int, v[2])
		for i := range indices {
			indices[i] = v[1] + i
		}
		return g.drawIndexed(v[0], indices)
	})
	for _, enabled := range []bool{true, false} {
		name := "glDisable"
		if enabled {
			name = "glEnable"
		}
		enabled := enabled
		ints(name, 1, func(v []int) error {
			switch v[0] {
			case glDepthTest:
				g.depthTest = enabled
			case glBlend:
				g.blend = enabled
			case glPolygonOffsetFill:
				g.polygonOffset = enabled
			case glLineSmooth:
				g.lineSmooth = enabled
			default:
				return py.ExceptionNewf(py.ValueError, "unsupported GL capability: %d", v[0])
			}
			return nil
		})
	}
	numbers("glClearColor", 4, func(v []float64) error {
		for i := 0; i < 4; i++ {
			g.clearColor[i] = math.Max(0, math.Min(1, v[i]))
		}
		return nil
	})
	ints("glClear", 1, func(v []int) error {
		mask := v[0]
		if mask & ^(glColorBit|glDepthBit) != 0 {
			return py.ExceptionNewf(py.ValueError, "unsupported clear mask")
		}
		if mask == glColorBit|glDepthBit {
			g.primitives = nil
			g.frameClear = g.clearColor
			return nil
		}
		if mask&glColorBit != 0 {
			return g.appendPrimitive(glPrimitive{mode: glVector{3}, colors: [3]glVector{g.clearColor}})
		}
		if mask&glDepthBit != 0 {
			return g.appendPrimitive(glPrimitive{mode: glVector{4}})
		}
		return nil
	})
	ints("glDepthFunc", 1, func(v []int) error {
		if v[0] != glLess && v[0] != glLequal && v[0] != glAlways {
			return py.ExceptionNewf(py.ValueError, "unsupported depth function")
		}
		g.depthFunc = v[0]
		return nil
	})
	numbers("glPolygonOffset", 2, func(v []float64) error { g.offsetFactor, g.offsetUnits = v[0], v[1]; return nil })
	numbers("glLineWidth", 1, func(v []float64) error {
		if v[0] <= 0 {
			return py.ExceptionNewf(py.ValueError, "line width must be positive")
		}
		g.lineWidth = v[0]
		return nil
	})
	ints("glBlendFunc", 2, func(v []int) error {
		if v[0] == 1 && v[1] == 0 {
			g.blendAlpha = false
			return nil
		}
		if v[0] != 0x0302 || v[1] != 0x0303 {
			return py.ExceptionNewf(py.ValueError, "only SRC_ALPHA/ONE_MINUS_SRC_ALPHA or ONE/ZERO blending is supported")
		}
		g.blendAlpha = true
		return nil
	})
	ints("glHint", 2, func(v []int) error {
		if v[0] != 0x0C52 || v[1] != 0x1102 {
			return py.ExceptionNewf(py.ValueError, "unsupported GL hint")
		}
		return nil
	})
	ints("glViewport", 4, func(v []int) error {
		if v[2] < 0 || v[3] < 0 {
			return py.ExceptionNewf(py.ValueError, "negative viewport dimensions")
		}
		copy(g.viewport[:], v)
		return nil
	})
	ints("glMatrixMode", 1, func(v []int) error {
		if v[0] != glProjection && v[0] != glModelview {
			return py.ExceptionNewf(py.ValueError, "unsupported matrix mode")
		}
		g.matrixMode = v[0]
		return nil
	})
	add("glLoadIdentity", 0, func(a py.Tuple) (py.Object, error) { *g.currentMatrix() = glIdentity(); return py.None, nil })
	numbers("glTranslatef", 3, func(v []float64) error {
		m := g.currentMatrix()
		*m = glMultiply(*m, glTranslation(v[0], v[1], v[2]))
		return nil
	})
	numbers("glRotatef", 4, func(v []float64) error {
		m := g.currentMatrix()
		*m = glMultiply(*m, glRotation(v[0], v[1], v[2], v[3]))
		return nil
	})
	ints("glBegin", 1, func(v []int) error {
		if g.inBegin {
			return py.ExceptionNewf(py.RuntimeError, "nested glBegin")
		}
		if v[0] != glTriangles && v[0] != glQuads && v[0] != glLines {
			return py.ExceptionNewf(py.ValueError, "unsupported primitive mode")
		}
		g.inBegin = true
		g.immediateMode = v[0]
		g.immediate = nil
		return nil
	})
	color := func(v []float64) error { g.currentColor = glVector{v[0], v[1], v[2], 1}; return nil }
	numbers("glColor3f", 3, color)
	add("glColor3fv", 1, func(a py.Tuple) (py.Object, error) {
		v, e := glFlat(a[0])
		if e != nil {
			return nil, e
		}
		if len(v) != 3 {
			return nil, py.ExceptionNewf(py.ValueError, "expected three color components")
		}
		return py.None, color(v)
	})
	vertex := func(v []float64) error {
		if !g.inBegin {
			return py.ExceptionNewf(py.RuntimeError, "vertex outside glBegin/glEnd")
		}
		g.immediate = append(g.immediate, glVertex{clip: glVector{v[0], v[1], v[2], 1}, color: g.currentColor})
		return nil
	}
	numbers("glVertex3f", 3, vertex)
	add("glVertex3fv", 1, func(a py.Tuple) (py.Object, error) {
		v, e := glFlat(a[0])
		if e != nil {
			return nil, e
		}
		if len(v) != 3 {
			return nil, py.ExceptionNewf(py.ValueError, "expected three position components")
		}
		return py.None, vertex(v)
	})
	add("glEnd", 0, func(a py.Tuple) (py.Object, error) {
		if !g.inBegin {
			return nil, py.ExceptionNewf(py.RuntimeError, "glEnd without glBegin")
		}
		g.inBegin = false
		matrix := glMultiply(g.projection, g.modelview)
		for i := range g.immediate {
			g.immediate[i].clip = matrix.apply(g.immediate[i].clip)
		}
		return py.None, g.draw(g.immediateMode, g.immediate)
	})
	add("glGetError", 0, func(a py.Tuple) (py.Object, error) { return py.Int(0), nil })
	add("glGetString", 1, func(a py.Tuple) (py.Object, error) {
		n, e := glInt(a[0])
		if e != nil {
			return nil, e
		}
		switch n {
		case 0x1F02:
			return py.Bytes("3.2 subset (Ebitengine emulation)"), nil
		case 0x1F01:
			return py.Bytes("Ebitengine Kage GL compatibility renderer"), nil
		case 0x1F00:
			return py.Bytes("web-computer"), nil
		}
		return nil, py.ExceptionNewf(py.ValueError, "unsupported GL string query")
	})
	ints("glDeleteShader", 1, func(v []int) error { delete(g.shaders, v[0]); return nil })
	ints("glDeleteProgram", 1, func(v []int) error {
		delete(g.programs, v[0])
		if g.program == v[0] {
			g.program = 0
		}
		return nil
	})
	py.RegisterModule(&py.ModuleImpl{Info: py.ModuleInfo{Name: "OpenGL", Doc: "Ebitengine OpenGL compatibility subset"}})
	py.RegisterModule(&py.ModuleImpl{Info: py.ModuleInfo{Name: "OpenGL.GL", Doc: "Stateful GL subset rendered by Ebitengine Kage"}, Globals: globals, Methods: methods})
	py.RegisterModule(&py.ModuleImpl{Info: py.ModuleInfo{Name: "OpenGL.GLU"}, Methods: []*py.Method{py.MustNewMethod("gluPerspective", func(self py.Object, a py.Tuple) (py.Object, error) {
		if len(a) != 4 {
			return nil, py.ExceptionNewf(py.TypeError, "gluPerspective expects four arguments")
		}
		v, e := glNumbers(a)
		if e != nil {
			return nil, e
		}
		if v[0] <= 0 || v[0] >= 180 || v[1] <= 0 || v[2] <= 0 || v[3] <= v[2] {
			return nil, py.ExceptionNewf(py.ValueError, "invalid perspective parameters")
		}
		m := g.currentMatrix()
		*m = glMultiply(*m, glPerspective(v[0], v[1], v[2], v[3]))
		return py.None, nil
	}, 0, "")}})
	registerGLDependencies()
}

func (g *glContext) fetchAttribute(loc, index int) (glVector, error) {
	out := glVector{0, 0, 0, 1}
	a, ok := g.vaos[g.vao].attrs[loc]
	if !ok || !a.enabled {
		return out, nil
	}
	bytes := g.buffers[a.buffer]
	size := 4
	if a.kind == glUnsignedByte {
		size = 1
	}
	stride := a.stride
	if stride == 0 {
		stride = a.size * size
	}
	if a.offset > len(bytes) || a.offset < 0 || index < 0 || index > (len(bytes)-a.offset)/stride {
		return out, py.ExceptionNewf(py.ValueError, "vertex index outside buffer")
	}
	offset := a.offset + index*stride
	if a.size*size > len(bytes)-offset {
		return out, py.ExceptionNewf(py.ValueError, "attribute range outside buffer")
	}
	for i := 0; i < a.size; i++ {
		if size == 4 {
			out[i] = float64(math.Float32frombits(binary.LittleEndian.Uint32(bytes[offset+i*4:])))
		} else {
			out[i] = float64(bytes[offset+i])
			if a.normalized {
				out[i] /= 255
			}
		}
	}
	return out, nil
}
func (g *glContext) drawIndexed(mode int, indices []int) error {
	p, ok := g.programs[g.program]
	if !ok || !p.linked {
		return py.ExceptionNewf(py.RuntimeError, "indexed drawing requires a linked active program")
	}
	matrix := glIdentity()
	if p.transforms {
		matrix = glMultiply(p.matrices["projection"], glMultiply(p.matrices["view"], p.matrices["model"]))
	}
	vertices := make([]glVertex, len(indices))
	for i, index := range indices {
		position, e := g.fetchAttribute(p.attrs["vertex_position"], index)
		if e != nil {
			return e
		}
		color, e := g.fetchAttribute(p.attrs["vertex_colour"], index)
		if e != nil {
			return e
		}
		if p.colorModifiers {
			mul, add := p.vectors["colour_mul"], p.vectors["colour_add"]
			for j := 0; j < 4; j++ {
				color[j] = color[j]*mul[j] + add[j]
			}
		}
		vertices[i] = glVertex{matrix.apply(position), color}
	}
	return g.draw(mode, vertices)
}
