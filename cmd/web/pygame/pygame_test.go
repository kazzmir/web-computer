package pygame

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kazzmir/web-computer/gpython/py"
	_ "github.com/kazzmir/web-computer/gpython/stdlib"
	"github.com/kazzmir/web-computer/scripts"
)

type testSystem struct{ frames int }

func (s *testSystem) Flip()                           { s.frames++ }
func (s *testSystem) AddDrawCall(func(*ebiten.Image)) {}
func (s *testSystem) SetScreenSize(int, int)          {}

type testGame struct{ t *testing.T }

func (g *testGame) Layout(int, int) (int, int) { return 1280, 480 }
func (g *testGame) Draw(*ebiten.Image)         {}
func (g *testGame) Update() error {
	ctx := py.NewContext(py.DefaultContextOpts())
	defer ctx.Close()
	system := &testSystem{}
	if err := RegisterPygameModule(system, ctx); err != nil {
		g.t.Error(err)
		return ebiten.Termination
	}
	script, err := scripts.Load("chimp.py")
	if err != nil {
		g.t.Fatal(err)
	}
	run := func(src, filename string, module interface{}) (*py.Module, error) {
		code, err := py.Compile(src, filename, "exec", 0, false)
		if err != nil {
			return nil, err
		}
		return py.RunCode(ctx, code, filename, module)
	}
	mod, err := run(string(script), "python/chimp.py", "chimp_test")
	if err != nil {
		g.t.Errorf("load chimp: %v", err)
		return ebiten.Termination
	}
	_, err = run(`
# Exercise resource loading, mouse positioning, walking, collisions, spin,
# transforms, transparency, group rendering, and the unmodified main loop.
pg.init()
screen = pg.display.set_mode((1280, 480), pg.SCALED)
chimp = Chimp()
fist = Fist()
assert chimp.image.get_size() == (244, 356)
assert fist.image.get_at((0, 0))[3] == 0
for i in range(180):
    chimp.update()
assert chimp.rect.width == 244
fist.rect = RectForTest = pg.Rect(chimp.rect.x, chimp.rect.y, 256, 236)
assert fist.punch(chimp)
fist.unpunch()
chimp.punched()
for i in range(30):
    chimp.update()
assert not chimp.dizzy
assert chimp.image is chimp.original
assert pg.transform.rotate(chimp.image, 90).get_size() == (356, 244)
assert pg.Rect(0, 0, 10, 10).inflate(-5, -5).topleft == (2, 2)
assert not pg.Rect(0, 0, 10, 10).colliderect(pg.Rect(10, 0, 1, 1))
assert pg.Rect(0, 0, 10, 10).contains(pg.Rect(1, 1, 2, 2))
r = chimp.image.get_rect(center=(300, 200))
assert r.center == (300, 200)

event_calls = 0
class TestEvent:
    def __init__(self, kind): self.type = kind

def test_events():
    global event_calls
    event_calls += 1
    if event_calls == 2: return [TestEvent(pg.MOUSEBUTTONDOWN)]
    if event_calls == 3: return [TestEvent(pg.MOUSEBUTTONUP)]
    if event_calls == 5: return [TestEvent(pg.QUIT)]
    return []
pg.event.get = test_events
main()
assert event_calls == 5
`, "chimp assertions", mod)
	if err != nil {
		g.t.Errorf("chimp integration: %v", err)
	}
	if system.frames != 6 {
		g.t.Errorf("got %d frames, want 6", system.frames)
	}
	g.t.Run("Scaletest", testScaletest)
	return ebiten.Termination
}

func TestChimp(t *testing.T) {
	if err := ebiten.RunGame(&testGame{t: t}); err != nil {
		t.Fatal(err)
	}
}
