package pygame

import (
	"testing"

	"github.com/kazzmir/web-computer/gpython/py"
	"github.com/kazzmir/web-computer/scripts"
)

// Runs inside testGame.Update: Ebitengine pixel reads require a running game.
func testScaletest(t *testing.T) {
	opts := py.DefaultContextOpts()
	opts.SysArgs = []string{"scaletest.py", "data/chimp.png", "-t", "-convert_alpha"}
	ctx := py.NewContext(opts)
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
	script, err := scripts.Load("scaletest.py")
	if err != nil {
		t.Fatal(err)
	}
	// Exercise the actual CLI entry point, including the speed-test flags.
	mod, err := run(string(script), "python/scaletest.py", py.MainModuleName)
	if err != nil {
		t.Fatalf("speed test: %v", err)
	}
	_, err = run(`
source = pg.Surface((2, 1))
source.fill((0, 0, 0))
white = pg.Surface((1, 1))
white.fill((255, 255, 255))
source.blit(white, (1, 0))
nearest = pg.transform.scale(source, (4, 1))
smooth = pg.transform.smoothscale(source, (4, 1))
assert nearest.get_at((1, 0))[0] == 0
assert nearest.get_at((2, 0))[0] == 255
assert 0 < smooth.get_at((1, 0))[0] < 255
assert 0 < smooth.get_at((2, 0))[0] < 255
for transform in (pg.transform.scale, pg.transform.smoothscale):
    for size in ((0, 0), (0, 7), (9, 0)):
        empty = transform(source, size)
        assert empty.get_size() == size
        assert empty.get_rect().size == size
        assert pg.transform.flip(empty, True, False).get_size() == size
        source.blit(empty, (0, 0))
        empty.fill((255, 255, 255))
    try:
        transform(source, (-1, 1))
    except ValueError:
        pass
    else:
        assert False

pg.display.init()
pg.mouse.set_visible(0)
pg.mouse.set_visible(1)

sizes = []
real_smoothscale = pg.transform.smoothscale
def record_scale(surface, size):
    sizes.append(tuple(size))
    return real_smoothscale(surface, size)
pg.transform.smoothscale = record_scale

calls = 0
class TestEvent:
    def __init__(self, kind, key):
        self.type = kind
        self.key = key

def test_events():
    global calls
    calls += 1
    if calls == 1:
        return [TestEvent(pg.KEYDOWN, pg.K_UP), TestEvent(pg.KEYDOWN, pg.K_LEFT)]
    if calls == 3:
        return [TestEvent(pg.KEYUP, pg.K_UP), TestEvent(pg.KEYUP, pg.K_LEFT),
                TestEvent(pg.KEYDOWN, pg.K_DOWN), TestEvent(pg.KEYDOWN, pg.K_RIGHT)]
    if calls == 4:
        return [TestEvent(pg.KEYUP, pg.K_DOWN), TestEvent(pg.KEYUP, pg.K_RIGHT)]
    if calls == 6:
        return [TestEvent(pg.KEYDOWN, pg.K_ESCAPE)]
    return []
pg.event.get = test_events
main('data/chimp.png', convert_alpha=True)
assert calls == 6
assert sizes == [(61, 89), (59, 87), (57, 85), (59, 87), (59, 87), (59, 87)]
assert pg.mouse.set_visible(True)
# Restore windowed mode for the remaining tests.
pg.display.set_mode((1280, 480))
`, "scaletest assertions", mod)
	if err != nil {
		t.Errorf("interactive scaler: %v", err)
	}
	if system.frames != 6 {
		t.Errorf("got %d frames, want 6", system.frames)
	}
}
