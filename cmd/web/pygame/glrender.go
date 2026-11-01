package pygame

import (
	_ "embed"
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kazzmir/web-computer/gpython/py"
)

//go:embed gl.kage
var glShaderSource string

// Geometry consumes seven vec4 uniforms per primitive. Keep room for
// Ebitengine's preserved uniforms within WebGL 2's minimum fragment budget.
// glcube uses 12 filled triangles and 12 edges per frame.
const glMaxPrimitives = 28

type glPrimitive struct {
	positions, colors [3]glVector
	mode              glVector
}

type glContext struct {
	shader                                                  *ebiten.Shader
	target                                                  *pixels
	viewport                                                [4]int
	primitives                                              []glPrimitive
	clearColor, frameClear, currentColor                    glVector
	depthTest, blend, blendAlpha, polygonOffset, lineSmooth bool
	depthFunc                                               int
	lineWidth, offsetFactor, offsetUnits                    float64
	projection, modelview                                   glMatrix
	matrixMode                                              int
	immediateMode                                           int
	immediate                                               []glVertex
	inBegin                                                 bool
	nextID                                                  int
	shaders                                                 map[int]*glShader
	programs                                                map[int]*glProgram
	buffers                                                 map[int][]byte
	vaos                                                    map[int]*glVAO
	uniforms                                                map[int]glUniform
	arrayBuffer, vao, program                               int
}

func newGLContext() (*glContext, error) {
	shader, e := ebiten.NewShader([]byte(glShaderSource))
	if e != nil {
		return nil, fmt.Errorf("GL emulator shader: %w", e)
	}
	g := &glContext{shader: shader, currentColor: glVector{1, 1, 1, 1}, lineWidth: 1, depthFunc: glLess, projection: glIdentity(), modelview: glIdentity(), matrixMode: glModelview, nextID: 1,
		shaders: map[int]*glShader{}, programs: map[int]*glProgram{}, buffers: map[int][]byte{}, vaos: map[int]*glVAO{0: {attrs: map[int]glAttribute{}}}, uniforms: map[int]glUniform{}}
	return g, nil
}
func (g *glContext) bindSurface(p *pixels, enabled bool) {
	g.target = nil
	if enabled {
		g.target = p
		size := p.bounds().Size()
		g.viewport = [4]int{0, 0, size.X, size.Y}
	}
	g.primitives = nil
	g.frameClear = g.clearColor
}
func (g *glContext) close() {
	g.shader.Deallocate()
	g.target, g.primitives, g.immediate = nil, nil, nil
	g.shaders, g.programs, g.buffers, g.vaos, g.uniforms = nil, nil, nil, nil, nil
}
func (g *glContext) render() error {
	if g.target == nil {
		return nil
	}
	positions, colors, modes := make([]float32, glMaxPrimitives*12), make([]float32, glMaxPrimitives*12), make([]float32, glMaxPrimitives*4)
	for i, p := range g.primitives {
		for j := 0; j < 3; j++ {
			for k := 0; k < 4; k++ {
				positions[i*12+j*4+k] = float32(p.positions[j][k])
				colors[i*12+j*4+k] = float32(p.colors[j][k])
			}
		}
		for k := 0; k < 4; k++ {
			modes[i*4+k] = float32(p.mode[k])
		}
	}
	clear := make([]float32, 4)
	for i, c := range g.frameClear {
		clear[i] = float32(c)
	}
	size := g.target.bounds().Size()
	g.target.image.DrawRectShader(size.X, size.Y, g.shader, &ebiten.DrawRectShaderOptions{Blend: ebiten.BlendCopy, Uniforms: map[string]any{
		"Positions": positions, "Colors": colors, "Modes": modes, "Count": len(g.primitives), "ClearColor": clear,
	}})
	return nil
}
func (g *glContext) appendPrimitive(p glPrimitive) error {
	if len(g.primitives) >= glMaxPrimitives {
		return py.ExceptionNewf(py.RuntimeError, "GL emulator supports at most %d clipped primitives between clears", glMaxPrimitives)
	}
	g.primitives = append(g.primitives, p)
	return nil
}
func (g *glContext) screenVertex(v glVertex) (glVector, glVector, bool) {
	if v.clip[3] <= 1e-9 {
		return glVector{}, glVector{}, false
	}
	inv := 1 / v.clip[3]
	for _, n := range v.clip {
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return glVector{}, glVector{}, false
		}
	}
	vp := g.viewport
	pos := glVector{float64(vp[0]) + (v.clip[0]*inv+1)*float64(vp[2])/2, float64(g.target.bounds().Dy()) - float64(vp[1]) - (v.clip[1]*inv+1)*float64(vp[3])/2, (v.clip[2]*inv + 1) / 2, inv}
	var color glVector
	for i := 0; i < 4; i++ {
		color[i] = v.color[i] * inv
	}
	return pos, color, true
}
func (g *glContext) primitiveMode(line bool) glVector {
	kind := 0.0
	if line {
		kind = 1
		if g.lineSmooth {
			kind = 2
		}
	}
	depth := 0.0
	if g.depthTest {
		depth = 1
		if g.depthFunc == glLequal {
			depth = 2
		}
		if g.depthFunc == glAlways {
			depth = 3
		}
	}
	blend := 0.0
	if g.blend && g.blendAlpha {
		blend = 1
	}
	return glVector{kind, depth, blend, g.lineWidth}
}
func (g *glContext) triangle(vertices []glVertex) error {
	clipped := clipGLPolygon(vertices)
	for i := 1; i+1 < len(clipped); i++ {
		p := glPrimitive{mode: g.primitiveMode(false)}
		valid := true
		for j, v := range []glVertex{clipped[0], clipped[i], clipped[i+1]} {
			var ok bool
			p.positions[j], p.colors[j], ok = g.screenVertex(v)
			valid = valid && ok
		}
		if !valid {
			continue
		}
		if g.polygonOffset {
			a, b, c := p.positions[0], p.positions[1], p.positions[2]
			dx1, dy1, dz1 := b[0]-a[0], b[1]-a[1], b[2]-a[2]
			dx2, dy2, dz2 := c[0]-a[0], c[1]-a[1], c[2]-a[2]
			determinant := dx1*dy2 - dy1*dx2
			slope := 0.0
			if math.Abs(determinant) > 1e-12 {
				slope = math.Max(math.Abs((dz1*dy2-dz2*dy1)/determinant), math.Abs((dx1*dz2-dx2*dz1)/determinant))
			}
			offset := g.offsetFactor*slope + g.offsetUnits/math.Pow(2, 24)
			for j := 0; j < 3; j++ {
				p.positions[j][2] += offset
			}
		}
		if e := g.appendPrimitive(p); e != nil {
			return e
		}
	}
	return nil
}
func (g *glContext) line(a, b glVertex) error {
	a, b, ok := clipGLLine(a, b)
	if !ok {
		return nil
	}
	p := glPrimitive{mode: g.primitiveMode(true)}
	var valid bool
	p.positions[0], p.colors[0], valid = g.screenVertex(a)
	if !valid {
		return nil
	}
	p.positions[1], p.colors[1], valid = g.screenVertex(b)
	if !valid {
		return nil
	}
	return g.appendPrimitive(p)
}
func (g *glContext) draw(mode int, vertices []glVertex) error {
	if g.target == nil {
		return py.ExceptionNewf(py.RuntimeError, "GL drawing requires an OPENGL display")
	}
	switch mode {
	case glTriangles:
		for i := 0; i+2 < len(vertices); i += 3 {
			if e := g.triangle(vertices[i : i+3]); e != nil {
				return e
			}
		}
	case glQuads:
		for i := 0; i+3 < len(vertices); i += 4 {
			for _, tri := range [][]glVertex{{vertices[i], vertices[i+1], vertices[i+2]}, {vertices[i], vertices[i+2], vertices[i+3]}} {
				if e := g.triangle(tri); e != nil {
					return e
				}
			}
		}
	case glLines:
		for i := 0; i+1 < len(vertices); i += 2 {
			if e := g.line(vertices[i], vertices[i+1]); e != nil {
				return e
			}
		}
	default:
		return py.ExceptionNewf(py.ValueError, "unsupported GL primitive mode: %d", mode)
	}
	return nil
}
func (g *glContext) currentMatrix() *glMatrix {
	if g.matrixMode == glProjection {
		return &g.projection
	}
	return &g.modelview
}
