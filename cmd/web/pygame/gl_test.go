package pygame

import (
	"image/color"
	"math"
	"testing"

	"github.com/kazzmir/web-computer/gpython/py"
	"github.com/kazzmir/web-computer/scripts"
)

func TestGLShaderContract(t *testing.T) {
	source := "void main() { vertex_color_out = vertex_colour; gl_Position = vec4(vertex_position, 1.0); }"
	if !shaderContract(&glShader{stage: glVertexShader, source: source}) {
		t.Error("supported shader rejected")
	}
	source = "void main() { vertex_color_out = vertex_colour; gl_Position = vec4(vertex_position, 1.0); gl_Position.x += 1.0; }"
	if shaderContract(&glShader{stage: glVertexShader, source: source}) {
		t.Error("unsupported shader operations silently accepted")
	}
}

func TestGLMatricesAndClipping(t *testing.T) {
	v := glMultiply(glTranslation(1, 2, 3), glRotation(90, 0, 1, 0)).apply(glVector{1, 0, 0, 1})
	for i, want := range (glVector{1, 2, 2, 1}) {
		if math.Abs(v[i]-want) > 1e-9 {
			t.Errorf("transformed coordinate %d = %f, want %f", i, v[i], want)
		}
	}
	clipped := clipGLPolygon([]glVertex{{clip: glVector{-0.5, -0.5, -2, 1}}, {clip: glVector{0.5, -0.5, 0, 1}}, {clip: glVector{0, 0.5, 0, 1}}})
	if len(clipped) != 4 {
		t.Fatalf("near-clipped triangle has %d vertices, want 4", len(clipped))
	}
	for _, vertex := range clipped {
		for plane := 0; plane < 6; plane++ {
			if glPlane(vertex, plane) < -1e-9 {
				t.Error("vertex outside clip volume")
			}
		}
	}
	if _, _, ok := clipGLLine(glVertex{clip: glVector{0, 0, -3, 1}}, glVertex{clip: glVector{1, 0, -2, 1}}); ok {
		t.Error("line behind near plane was not rejected")
	}
}

// Called in the Ebitengine game so Kage execution and pixel readback are legal.
func testGLRenderer(t *testing.T) {
	g, err := newGLContext()
	if err != nil {
		t.Fatal(err)
	}
	defer g.close()
	surface := newPixels(32, 32)
	g.bindSurface(surface, true)
	g.frameClear = glVector{0, 0, 0, 1}
	g.depthTest = true
	red, blue := glVector{1, 0, 0, 1}, glVector{0, 0, 1, 1}
	triangle := func(z float64, c glVector) []glVertex {
		return []glVertex{{glVector{-0.9, -0.9, z, 1}, c}, {glVector{0.9, -0.9, z, 1}, c}, {glVector{0, 0.9, z, 1}, c}}
	}
	sample := func() color.RGBA {
		if e := g.render(); e != nil {
			t.Fatal(e)
		}
		return color.RGBAModel.Convert(surface.image.At(16, 16)).(color.RGBA)
	}
	for _, reverse := range []bool{false, true} {
		g.primitives = nil
		near, far := triangle(-0.5, red), triangle(0.5, blue)
		if reverse {
			near, far = far, near
		}
		if e := g.draw(glTriangles, near); e != nil {
			t.Fatal(e)
		}
		if e := g.draw(glTriangles, far); e != nil {
			t.Fatal(e)
		}
		if c := sample(); c.R < 250 || c.B > 5 {
			t.Errorf("depth testing (reverse %v): got %v, want red", reverse, c)
		}
	}
	g.depthTest = false
	g.primitives = nil
	g.draw(glTriangles, triangle(-0.5, red))
	g.draw(glTriangles, triangle(0.5, blue))
	if c := sample(); c.B < 250 || c.R > 5 {
		t.Errorf("disabled depth test: got %v, want blue", c)
	}
	// Hidden edges must not draw over a nearer face.
	g.depthTest = true
	g.primitives = nil
	g.draw(glTriangles, triangle(-0.5, red))
	g.draw(glLines, []glVertex{{glVector{-0.7, 0, 0.5, 1}, glVector{1, 1, 1, 1}}, {glVector{0.7, 0, 0.5, 1}, glVector{1, 1, 1, 1}}})
	if c := sample(); c.R < 250 || c.G > 5 {
		t.Errorf("hidden edge: got %v, want red", c)
	}
	// Partial clears must preserve the buffer they do not clear.
	g.primitives = nil
	g.draw(glTriangles, triangle(-0.5, red))
	g.appendPrimitive(glPrimitive{mode: glVector{4}}) // depth clear
	g.draw(glTriangles, triangle(0.5, blue))
	if c := sample(); c.B < 250 {
		t.Errorf("depth-only clear: got %v, want blue", c)
	}
	g.primitives = nil
	g.draw(glTriangles, triangle(-0.5, red))
	g.appendPrimitive(glPrimitive{mode: glVector{3}, colors: [3]glVector{{0, 1, 0, 1}}}) // color clear
	g.draw(glTriangles, triangle(0.5, blue))
	if c := sample(); c.G < 250 || c.B > 5 {
		t.Errorf("color-only clear lost depth: %v", c)
	}

	g.primitives = nil
	g.depthTest, g.blend, g.blendAlpha = false, true, true
	g.frameClear = blue
	g.draw(glTriangles, triangle(0, glVector{1, 0, 0, 0.5}))
	if c := sample(); math.Abs(float64(c.R)-96) > 2 || math.Abs(float64(c.B)-96) > 2 || math.Abs(float64(c.A)-191) > 2 {
		t.Errorf("alpha blending: got %v", c)
	}
	g.depthTest, g.blend = true, false
	g.frameClear = glVector{0, 0, 0, 1}

	// Colors must be perspective-correct, not affine screen-space interpolation.
	g.primitives = nil
	vertices := []glVertex{{glVector{-0.8, -0.8, 0, 1}, red}, {glVector{1.6, -1.6, 0, 2}, glVector{0, 1, 0, 1}}, {glVector{0, 3.2, 0, 4}, blue}}
	if e := g.draw(glTriangles, vertices); e != nil {
		t.Fatal(e)
	}
	c := sample()
	// Calculate barycentric weights at pixel center (16.5,16.5).
	a, b, d := g.primitives[0].positions[0], g.primitives[0].positions[1], g.primitives[0].positions[2]
	cross := func(x, y, z, w float64) float64 { return x*w - y*z }
	area := cross(b[0]-a[0], b[1]-a[1], d[0]-a[0], d[1]-a[1])
	wa := cross(b[0]-16.5, b[1]-16.5, d[0]-16.5, d[1]-16.5) / area
	wb := cross(d[0]-16.5, d[1]-16.5, a[0]-16.5, a[1]-16.5) / area
	wc := 1 - wa - wb
	total := wa + wb/2 + wc/4
	expected := []float64{255 * wa / total, 255 * wb / 2 / total, 255 * wc / 4 / total}
	for i, got := range []uint8{c.R, c.G, c.B} {
		if math.Abs(float64(got)-expected[i]) > 2 {
			t.Errorf("perspective color %d = %d, want %.2f", i, got, expected[i])
		}
	}
}

func testGLCube(t *testing.T) {
	ctx := py.NewContext(py.DefaultContextOpts())
	defer ctx.Close()
	system := &testSystem{}
	if err := RegisterPygameModule(system, ctx); err != nil {
		t.Fatal(err)
	}
	run := func(src, filename string, module interface{}) (*py.Module, error) {
		code, err := py.Compile(src, filename, "exec", 0, false)
		if err != nil {
			return nil, err
		}
		return py.RunCode(ctx, code, filename, module)
	}
	script, err := scripts.Load("glcube.py")
	if err != nil {
		t.Fatal(err)
	}
	mod, err := run(string(script), "python/glcube.py", "glcube_test")
	if err != nil {
		t.Fatalf("load glcube: %v", err)
	}
	_, err = run(`
assert ctypes.c_void_p(12).value == 12
m = eye(4, dtype=float32)
translate(m, 2, 3, -6)
assert m[3, 0] == 2 and m[3, 1] == 3 and m[3, 2] == -6
assert dot(eye(4), m).tolist() == m.tolist()
assert array([1, 2, 3], dtype=uint32).nbytes == 12
pg.display.gl_set_attribute(pg.GL_CONTEXT_MAJOR_VERSION, 3)
assert pg.display.gl_get_attribute(pg.GL_CONTEXT_MAJOR_VERSION) == 3
screen = pg.display.set_mode((64, 64), pg.OPENGL | pg.DOUBLEBUF)
gpu, filled, outline = init_gl_modern((64, 64))
assert filled.nbytes == len(filled)*4
assert GL.glGetUniformLocation(1, 'missing') == -1
rotation = Rotation()
frames = []
for i in range(3):
    draw_cube_modern(gpu, filled, outline, rotation)
    pg.display.flip()
    frames.append([screen.get_at((x, y)) for y in range(64) for x in range(64)])
assert frames[0] != frames[1]
assert frames[1] != frames[2]
assert sum([1 for c in frames[1] if max(c[:3]) > 0]) > 200
assert sum([1 for c in frames[1] if min(c[:3]) > 235]) > 0
assert sum([1 for c in frames[1] if max(c[:3]) != min(c[:3])]) > 100
assert rotation.phi == 43
assert pg.time.wait(0) >= 0
pg.time.wait = lambda milliseconds: milliseconds

# Compile failures are visible, not pretend-success GLSL compilation.
bad = GL.glCreateShader(GL.GL_VERTEX_SHADER)
GL.glShaderSource(bad, 'this is not a supported shader')
GL.glCompileShader(bad)
assert GL.glGetShaderiv(bad, GL.GL_COMPILE_STATUS) == 0
assert len(GL.glGetShaderInfoLog(bad)) > 0
GL.glDeleteShader(bad)
try:
    GL.glColor3fv('not numeric data')
except TypeError:
    pass
else:
    assert False
try:
    GL.glDrawElements(GL.GL_TRIANGLES, 999999, GL.GL_UNSIGNED_INT, None)
except ValueError:
    pass
else:
    assert False

# The unchanged main loop must reinitialize its buffers and uniforms across
# both fullscreen transitions, and exit cleanly on Escape.
calls = 0
class TestEvent:
    def __init__(self, kind, key=None): self.type, self.key = kind, key
def modern_events():
    global calls
    calls += 1
    if calls == 2 or calls == 3: return [TestEvent(pg.KEYDOWN, pg.K_f)]
    if calls == 4: return [TestEvent(pg.KEYDOWN, pg.K_ESCAPE)]
    return []
pg.event.get = modern_events
main()
assert calls == 4
assert pg.display.gl_get_attribute(pg.GL_CONTEXT_MINOR_VERSION) == 2

# Exercise the second code path too: glBegin/glEnd, quads, colored vertices,
# projection/modelview matrices, GLU perspective, and white wireframe edges.
USE_MODERN_GL = False
calls = 0
def legacy_events():
    global calls
    calls += 1
    return [TestEvent(pg.QUIT)] if calls == 3 else []
pg.event.get = legacy_events
main()
assert calls == 3
screen = pg.display.get_surface()
assert sum([1 for y in range(160, 320, 4) for x in range(200, 440, 4) if max(screen.get_at((x, y))[:3]) > 0]) > 100
assert pg.mouse.set_visible(True)
`, "glcube assertions", mod)
	if err != nil {
		t.Fatalf("glcube integration: %v", err)
	}
	if system.frames != 10 {
		t.Errorf("got %d frames, want 10", system.frames)
	}
}
