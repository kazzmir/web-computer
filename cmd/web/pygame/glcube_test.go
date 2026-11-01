package pygame

import (
	"testing"

	_ "github.com/kazzmir/web-computer/gpython/lib"
	"github.com/kazzmir/web-computer/gpython/py"
	"github.com/kazzmir/web-computer/scripts"
)

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
