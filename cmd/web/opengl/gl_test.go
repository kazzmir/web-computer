package opengl

import (
	"image/color"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
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

type rendererTestGame struct{ t *testing.T }

func (*rendererTestGame) Layout(int, int) (int, int) { return 32, 32 }
func (*rendererTestGame) Draw(*ebiten.Image)         {}
func (game *rendererTestGame) Update() error {
	testGLRenderer(game.t)
	return ebiten.Termination
}
func TestGLRenderer(t *testing.T) {
	if err := ebiten.RunGame(&rendererTestGame{t: t}); err != nil {
		t.Fatal(err)
	}
}

// Called in the Ebitengine game so Kage execution and pixel readback are legal.
func testGLRenderer(t *testing.T) {
	g, err := NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	surface := ebiten.NewImage(32, 32)
	g.BindSurface(surface)
	g.frameClear = glVector{0, 0, 0, 1}
	g.depthTest = true
	red, blue := glVector{1, 0, 0, 1}, glVector{0, 0, 1, 1}
	triangle := func(z float64, c glVector) []glVertex {
		return []glVertex{{glVector{-0.9, -0.9, z, 1}, c}, {glVector{0.9, -0.9, z, 1}, c}, {glVector{0, 0.9, z, 1}, c}}
	}
	sample := func() color.RGBA {
		if e := g.Render(); e != nil {
			t.Fatal(e)
		}
		return color.RGBAModel.Convert(surface.At(16, 16)).(color.RGBA)
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
	g.appendPrimitive(glPrimitive{mode: glVector{4}})
	g.draw(glTriangles, triangle(0.5, blue))
	if c := sample(); c.B < 250 {
		t.Errorf("depth-only clear: got %v, want blue", c)
	}
	g.primitives = nil
	g.draw(glTriangles, triangle(-0.5, red))
	g.appendPrimitive(glPrimitive{mode: glVector{3}, colors: [3]glVector{{0, 1, 0, 1}}})
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
