package pygame

import (
	"testing"

	"github.com/kazzmir/web-computer/gpython/py"
	"github.com/kazzmir/web-computer/scripts"
)

// Invoked inside the shared Ebitengine test game so pixel readback is legal.
func testScroll(t *testing.T) {
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
	script, err := scripts.Load("scroll.py")
	if err != nil {
		t.Fatal(err)
	}
	mod, err := run(string(script), "python/scroll.py", "scroll_test")
	if err != nil {
		t.Fatalf("load scroll: %v", err)
	}
	_, err = run(`
assert scale is pg.transform.scale
assert pg.Color('beige') == (245, 245, 220, 255)
assert pg.Color('black') == (0, 0, 0, 255)
r = pg.Rect(1, 2, 3, 4)
r2 = r.copy()
r2.w, r2.h = 10, 12
assert r.size == (3, 4) and r2.size == (10, 12)

# Clips apply to fills and blits, and are returned by value.
s = pg.Surface((8, 6), 0, 24)
s.fill('beige')
s.set_clip((2, 1, 4, 3))
s.get_clip().w = 1
assert s.get_clip().size == (4, 3)
s.fill('black')
assert s.get_at((1, 1)) == pg.Color('beige')
assert s.get_at((2, 1)) == pg.Color('black')
white = pg.Surface((8, 6))
white.fill('white')
s.blit(white, (0, 0))
assert s.get_at((2, 1)) == pg.Color('white')
assert s.get_at((6, 1)) == pg.Color('beige')
s.set_clip((-1, -1, 3, 3))
assert s.get_clip().topleft == (0, 0) and s.get_clip().size == (2, 2)
s.set_clip(None)
assert s.get_clip().size == (8, 6)

# Subsurfaces share storage, have local coordinates, and own their clips.
sub = s.subsurface((2, 1, 4, 3))
assert sub.get_size() == (4, 3) and sub.get_clip().topleft == (0, 0)
sub.fill((12, 34, 56))
assert s.get_at((2, 1)) == sub.get_at((0, 0)) == (12, 34, 56, 255)
nested = sub.subsurface((1, 1, 2, 1))
nested.blit(white, (0, 0))
assert s.get_at((3, 2)) == nested.get_at((0, 0)) == pg.Color('white')
s.set_clip((0, 0, 1, 1))
sub.fill('black')
assert s.get_at((2, 1)) == pg.Color('black')
s.set_clip(None)
try:
    s.subsurface((-1, 0, 2, 2))
except ValueError:
    pass
else:
    assert False

# Destination-surface scaling writes through to the parent, not a copy.
source = pg.Surface((1, 1))
source.fill((50, 60, 70))
assert scale(source, nested.get_size(), nested) is nested
assert s.get_at((3, 2)) == (50, 60, 70, 255)
assert s.get_at((4, 2)) == (50, 60, 70, 255)
assert s.get_at((5, 2)) == pg.Color('black')
try:
    scale(source, (3, 1), nested)
except ValueError:
    pass
else:
    assert False

# Every direction must match a fresh rendering of the moved viewport.
image = pg.image.load(os.path.join(main_dir, 'data', 'arraydemo.bmp')).convert()
assert image.get_size() == (200, 128)
screen = pg.display.set_mode((40, 32))
for direction in (DIR_UP, DIR_DOWN, DIR_LEFT, DIR_RIGHT):
    screen.set_clip(None)
    screen.fill('beige')
    screen.set_clip((4, 4, 32, 24))
    view = pg.Rect(1, 1, 4, 3)
    scale(image.subsurface(view), (32, 24), screen.subsurface(screen.get_clip()))
    scroll_view(screen, image, direction, view)
    expected = scale(image.subsurface(view), (32, 24))
    for y in range(24):
        for x in range(32):
            assert screen.get_at((x+4, y+4)) == expected.get_at((x, y))
    assert screen.get_at((3, 10)) == pg.Color('beige')
    assert screen.get_at((36, 10)) == pg.Color('beige')
    assert screen.get_at((10, 3)) == pg.Color('beige')
    assert screen.get_at((10, 28)) == pg.Color('beige')

# View boundaries must not move beyond the source image.
for direction, rect in ((DIR_UP, (0, 0, 4, 3)), (DIR_LEFT, (0, 0, 4, 3)),
                        (DIR_DOWN, (0, 125, 4, 3)), (DIR_RIGHT, (196, 0, 4, 3))):
    view = pg.Rect(rect)
    original_pos = view.topleft
    scroll_view(screen, image, direction, view)
    assert view.topleft == original_pos

# Scrolling exposed pixels leaves them unchanged; even overlapping scrolls
# copy the old values rather than reading pixels already overwritten.
line = pg.Surface((5, 1))
for x in range(5):
    source.fill((x+1, 0, 0))
    line.blit(source, (x, 0))
line.scroll(dx=1)
assert [line.get_at((x, 0))[0] for x in range(5)] == [1, 1, 2, 3, 4]
line.scroll(dx=100)
assert [line.get_at((x, 0))[0] for x in range(5)] == [1, 1, 2, 3, 4]
line.set_clip((1, 0, 3, 1))
line.scroll(dx=-1)
assert [line.get_at((x, 0))[0] for x in range(5)] == [1, 2, 3, 3, 4]

parent = pg.Surface((6, 3))
parent.fill('beige')
row = parent.subsurface((1, 1, 4, 1))
for x in range(4):
    source.fill((x+1, 0, 0))
    row.blit(source, (x, 0))
row.scroll(dx=-1)
assert [parent.get_at((x, 1))[0] for x in range(6)] == [245, 2, 3, 4, 4, 245]

# Polygon hit regions must have exact direction colors, including on a view.
regions = pg.Surface((100, 100), 0, 24)
add_arrow_button(regions, regions, (50, 50), DIR_RIGHT)
assert regions.get_at((50, 50))[0] == DIR_RIGHT
assert regions.get_at((5, 5))[0] == 0
pg.draw.polygon(regions.subsurface((10, 10, 20, 20)), 'red', ((0, 0), (20, 0), (10, 20)))
assert regions.get_at((20, 15)) == pg.Color('red')

# Real input state APIs are present, before replacing them for the bounded loop.
assert len(pg.key.get_pressed()) > pg.K_RIGHT
pg.key.set_repeat(120, 60)
assert pg.key.get_repeat() == (120, 60)
keys = [False] * len(pg.key.get_pressed())
pg.key.get_pressed = lambda: keys
calls = 0
class TestEvent:
    def __init__(self, kind, key=None, pos=None):
        self.type, self.key, self.pos = kind, key, pos

def test_events():
    global calls
    calls += 1
    for key in (pg.K_UP, pg.K_DOWN, pg.K_LEFT, pg.K_RIGHT): keys[key] = False
    if calls == 1:
        keys[pg.K_DOWN] = keys[pg.K_RIGHT] = True
    elif calls == 2:
        return [TestEvent(pg.MOUSEBUTTONDOWN, pos=(200, 280))]
    elif calls == 4:
        return [TestEvent(pg.MOUSEBUTTONUP, pos=(200, 280))]
    elif calls == 5:
        keys[pg.K_UP] = keys[pg.K_LEFT] = True
    elif calls == 6:
        return [TestEvent(pg.KEYDOWN, key=pg.K_ESCAPE)]
    return []
pg.event.get = test_events
motions = []
real_scroll = scroll_view
def record_scroll(screen, image, direction, view):
    real_scroll(screen, image, direction, view)
    motions.append(view.topleft)
scroll_view = record_scroll
main()
assert calls == 6
assert motions == [(0, 1), (1, 1), (1, 2), (1, 3), (1, 2), (0, 2)]
assert pg.key.get_repeat() == (120, 60)
assert pg.mouse.set_visible(True)
`, "scroll assertions", mod)
	if err != nil {
		t.Fatalf("scroll integration: %v", err)
	}
	// Idle frames must yield and retain the display; ticks immediately following
	// a flip must not yield a second time (regression coverage for Chimp pacing).
	before := system.frames
	_, err = run("pg.display.flip()\nclock = pg.time.Clock()\nclock.tick(30)\nclock.tick(30)\n", "idle assertions", mod)
	if err != nil {
		t.Fatal(err)
	}
	if got := system.frames - before; got != 2 {
		t.Errorf("flip plus two ticks produced %d frames, want 2", got)
	}
}
