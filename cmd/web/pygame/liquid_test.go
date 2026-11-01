package pygame

import (
	"encoding/binary"
	"testing"

	"github.com/kazzmir/web-computer/gpython/py"
	"github.com/kazzmir/web-computer/scripts"
)

func TestScale2xRGBA(t *testing.T) {
	for _, tc := range []struct {
		name       string
		b, d, f, h uint32
		want       [4]uint32
	}{
		{"top left", 1, 1, 3, 4, [4]uint32{1, 2, 2, 2}},
		{"top right", 3, 1, 3, 4, [4]uint32{2, 3, 2, 2}},
		{"bottom left", 3, 1, 4, 1, [4]uint32{2, 2, 1, 2}},
		{"bottom right", 3, 1, 4, 4, [4]uint32{2, 2, 2, 4}},
		{"vertical match prevents smoothing", 1, 1, 3, 1, [4]uint32{2, 2, 2, 2}},
		{"horizontal match prevents smoothing", 1, 1, 1, 3, [4]uint32{2, 2, 2, 2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values := []uint32{9, tc.b, 9, tc.d, 2, tc.f, 9, tc.h, 9}
			data := make([]byte, len(values)*4)
			for i, v := range values {
				binary.LittleEndian.PutUint32(data[i*4:], v)
			}
			got := scale2xRGBA(data, 3, 3)
			for i, pos := range [][2]int{{2, 2}, {3, 2}, {2, 3}, {3, 3}} {
				if value := binary.LittleEndian.Uint32(got[(pos[1]*6+pos[0])*4:]); value != tc.want[i] {
					t.Errorf("quadrant %d = %d, want %d", i, value, tc.want[i])
				}
			}
		})
	}
	// Single-pixel edges replicate exactly, including all alpha bits.
	input := []byte{12, 34, 56, 78}
	got := scale2xRGBA(input, 1, 1)
	for i := 0; i < 4; i++ {
		if binary.LittleEndian.Uint32(got[i*4:]) != binary.LittleEndian.Uint32(input) {
			t.Error("single pixel was not preserved")
		}
	}
}

// Pixel operations run inside the shared Ebitengine game, like the other demos.
func testLiquid(t *testing.T) {
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
	script, err := scripts.Load("liquid.py")
	if err != nil {
		t.Fatal(err)
	}
	mod, err := run(string(script), "python/liquid.py", "liquid_test")
	if err != nil {
		t.Fatal(err)
	}
	_, err = run(`
from pygame.transform import scale2x
assert pg.Surface((1, 1)).get_bitsize() == 32
assert pg.display.set_mode((20, 20), pg.HWSURFACE | pg.DOUBLEBUF).get_bitsize() == 32

# Region blits must truncate fractional rectangles, clip both ends, and return
# the actual changed rectangle. Liquid supplies floating source coordinates.
source = pg.Surface((5, 3))
tile = pg.Surface((1, 1))
for y in range(3):
    for x in range(5):
        tile.fill((x*30, y*70, 11))
        source.blit(tile, (x, y))
dest = pg.Surface((9, 7))
dest.fill('beige')
r = dest.blit(source, (2, 3), (1.9, 1.2, 2, 1))
assert r.topleft == (2, 3) and r.size == (2, 1)
assert dest.get_at((2, 3)) == (30, 70, 11, 255)
assert dest.get_at((3, 3)) == (60, 70, 11, 255)
assert dest.get_at((4, 3)) == pg.Color('beige')
source.set_clip((0, 0, 1, 1))
assert dest.blit(source, pg.Rect(0, 0, 100, 100), pg.Rect(2, 2, 1, 1)).size == (1, 1)
assert dest.get_at((0, 0)) == (60, 140, 11, 255)
source.set_clip(None)
r = dest.blit(source, (1, 1), (-1, -1, 3, 3))
assert r.topleft == (2, 2) and r.size == (2, 2)
assert dest.get_at((2, 2)) == source.get_at((0, 0))
assert dest.get_at((3, 3)) == source.get_at((1, 1))
dest.fill('beige')
dest.set_clip((3, 2, 2, 2))
r = dest.blit(source, (2, 1), (0, 0, 5, 3))
assert r.topleft == (3, 2) and r.size == (2, 2)
assert dest.get_at((3, 2)) == source.get_at((1, 1))
assert dest.get_at((4, 3)) == source.get_at((2, 2))
assert dest.get_at((2, 2)) == pg.Color('beige')
dest.set_clip(None)
r = dest.blit(source, (-1, -1), (1, 0, 3, 3))
assert r.topleft == (0, 0) and r.size == (2, 2)
assert dest.get_at((0, 0)) == source.get_at((2, 1))
view = dest.subsurface((2, 2, 4, 3))
view.blit(source.subsurface((1, 0, 3, 3)), (1, 1), (1, 1, 2, 1))
assert dest.get_at((3, 3)) == source.get_at((2, 1))
assert dest.blit(source, (0, 0), (99, 99, 2, 2)).size == (0, 0)
assert dest.blit(source, (0, 0), (0, 0, -1, 1)).size == (0, 0)

# Self-blits and overlapping shared views must read a snapshot, not panic.
line = pg.Surface((5, 1))
for x in range(5):
    tile.fill((x+1, 0, 0))
    line.blit(tile, (x, 0))
line.blit(line, (1, 0), (0, 0, 4, 1))
assert [line.get_at((x, 0))[0] for x in range(5)] == [1, 1, 2, 3, 4]
line.blit(line.subsurface((0, 0, 3, 1)), (2, 0))
assert [line.get_at((x, 0))[0] for x in range(5)] == [1, 1, 1, 1, 2]

# This pattern distinguishes real Scale2x from ordinary nearest scaling.
pattern = pg.Surface((3, 3))
values = [9, 1, 9, 1, 2, 3, 9, 4, 9]
for y in range(3):
    for x in range(3):
        tile.fill((values[y*3+x], 0, 0))
        pattern.blit(tile, (x, y))
doubled = scale2x(pattern)
assert doubled.get_size() == (6, 6)
assert doubled.get_at((2, 2))[0] == 1
assert doubled.get_at((3, 2))[0] == 2
assert pg.transform.scale(pattern, (6, 6)).get_at((2, 2))[0] == 2
parent = pg.Surface((8, 8))
parent.fill('beige')
view = parent.subsurface((1, 1, 6, 6))
assert scale2x(pattern, view) is view
assert parent.get_at((3, 3))[0] == 1
assert parent.get_at((0, 0)) == pg.Color('beige')
assert scale2x(pattern.subsurface((1, 1, 1, 1))).get_size() == (2, 2)
assert scale2x(pg.Surface((0, 3))).get_size() == (0, 6)
try:
    scale2x(pattern, pg.Surface((3, 3)))
except ValueError:
    pass
else:
    assert False

# Run the unchanged liquid loop for two frames, then exit. Verify samples on
# both sides of tile boundaries against its sine-wave source coordinates.
bitmap = pg.image.load(os.path.join(main_dir, 'data', 'liquid.bmp'))
assert bitmap.get_size() == (172, 132)
bitmap = scale2x(scale2x(bitmap))
assert bitmap.get_size() == (688, 528)
samples = [(x, y) for x in (0, 19, 20, 321, 639) for y in (0, 19, 20, 301, 479)]
frames = []
real_flip = pg.display.flip
def record_frame():
    real_flip()
    screen = pg.display.get_surface()
    frames.append([screen.get_at(pos) for pos in samples])
pg.display.flip = record_frame
calls = 0
class StopEvent:
    def __init__(self, kind): self.type = kind

def test_events():
    global calls
    calls += 1
    return [StopEvent(pg.QUIT)] if calls == 3 else []
pg.event.get = test_events
main()
assert len(frames) == 2 and calls == 3
for i in range(2):
    anim = (i+1)*0.02
    for j in range(len(samples)):
        x, y = samples[j]
        block_x, block_y = (x//20)*20, (y//20)*20
        sx = int(block_x + sin(anim + block_x*0.01)*15 + 20) + x%20
        sy = int(block_y + sin(anim + block_y*0.01)*15 + 20) + y%20
        assert frames[i][j] == bitmap.get_at((sx, sy))
for kind in (pg.KEYDOWN, pg.MOUSEBUTTONDOWN):
    pg.event.get = lambda: [StopEvent(kind)]
    main()
assert len(frames) == 2
pg.quit()
`, "liquid assertions", mod)
	if err != nil {
		t.Fatalf("liquid integration: %v", err)
	}
	if system.frames != 2 {
		t.Errorf("got %d frames, want 2", system.frames)
	}
}
