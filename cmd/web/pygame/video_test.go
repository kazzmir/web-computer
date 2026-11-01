package pygame

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kazzmir/web-computer/gpython/py"
	"github.com/kazzmir/web-computer/scripts"
)

func testSDLDesktop(t *testing.T) {
	system := &testSystem{}
	d := newSDLDesktop(system, system.Flip)
	defer d.close()
	a := &sdlWindow{id: 1, owner: py.String("primary"), title: "Primary", position: image.Pt(12, 12), surface: newPixels(100, 100), opacity: 1, resizable: true}
	b := &sdlWindow{id: 2, owner: py.String("secondary"), title: "Secondary", position: image.Pt(32, 32), surface: newPixels(64, 64), opacity: 0.5, top: true}
	a.surface.image.Fill(color.RGBA{255, 0, 0, 255})
	b.surface.image.Fill(color.RGBA{0, 255, 0, 255})
	d.windows = []*sdlWindow{a, b}
	d.primary, d.focused = a, a
	if err := d.present(); err != nil {
		t.Fatal(err)
	}
	c := color.RGBAModel.Convert(d.scene.At(50, 80)).(color.RGBA)
	if c.R < 126 || c.R > 129 || c.G < 126 || c.G > 129 || c.B != 0 {
		t.Errorf("window opacity compositing: %v", c)
	}
	d.focus(a)
	if err := d.present(); err != nil {
		t.Fatal(err)
	}
	if c2 := color.RGBAModel.Convert(d.scene.At(50, 80)).(color.RGBA); c2 != c {
		t.Errorf("always-on-top window was hidden: %v", c2)
	}
	d.input = func() sdlInput { return sdlInput{mouse: image.Pt(20, 15), pressed: true, down: true} }
	d.poll()
	d.input = func() sdlInput { return sdlInput{mouse: image.Pt(40, 35), down: true} }
	d.poll()
	if a.position != image.Pt(32, 32) {
		t.Errorf("drag position: %v", a.position)
	}
	d.input = func() sdlInput {
		return sdlInput{mouse: b.closeRect().Min.Add(image.Pt(3, 3)), pressed: true, down: true}
	}
	events := d.poll()
	if len(events) != 1 {
		t.Fatalf("close events = %v", events)
	}
	event := events[0].(py.Tuple)
	if event[0] != py.Int(windowCloseEvent) || event[2] != b.owner {
		t.Errorf("wrong close event: %v", event)
	}
	d.destroy(b)
	if d.owner() != a.owner {
		t.Error("focus was not restored after closing secondary")
	}
	if err := d.present(); err != nil {
		t.Fatal(err)
	}
	if c := color.RGBAModel.Convert(d.scene.At(50, 80)).(color.RGBA); c.R != 255 || c.G != 0 {
		t.Errorf("destroyed window still visible: %v", c)
	}
	corner := a.bounds().Max.Sub(image.Pt(2, 2))
	d.input = func() sdlInput { return sdlInput{mouse: corner, pressed: true, down: true} }
	d.poll()
	d.input = func() sdlInput { return sdlInput{mouse: corner.Add(image.Pt(20, 10)), down: true} }
	d.poll()
	if a.surface.bounds().Size() != image.Pt(120, 110) {
		t.Errorf("resize = %v", a.surface.bounds())
	}
	if c := color.RGBAModel.Convert(a.surface.image.At(20, 20)).(color.RGBA); c.R != 255 {
		t.Errorf("resize lost window contents: %v", c)
	}
	d.input = func() sdlInput { return sdlInput{} }
	d.poll()
	// Exercise real dialog drawing and keyboard selection without blocking CI.
	inputs := 0
	d.input = func() sdlInput {
		inputs++
		if inputs == 1 {
			return sdlInput{keys: []ebiten.Key{ebiten.KeyRight}}
		}
		return sdlInput{keys: []ebiten.Key{ebiten.KeyEnter}}
	}
	answer, err := d.messagebox("Question", "Continue?", []string{"Yes", "No", "Chance"}, 0, 1)
	if err != nil || answer != 1 {
		t.Fatalf("dialog selection = %d, %v", answer, err)
	}
	d.input = func() sdlInput { return sdlInput{keys: []ebiten.Key{ebiten.KeyEscape}} }
	answer, err = d.messagebox("Question", "Continue?", []string{"Yes", "No"}, 0, 1)
	if err != nil || answer != 1 {
		t.Fatalf("dialog Escape = %d, %v", answer, err)
	}
	cancel := newSDLDesktop(system, func() error { return py.ExceptionNewf(py.SystemExit, "closed") })
	defer cancel.close()
	cancel.input = func() sdlInput { return sdlInput{} }
	_, err = cancel.messagebox("Question", "Continue?", []string{"OK"}, 0, 0)
	if !py.IsException(py.SystemExit, err) || !py.IsException(py.SystemExit, cancel.check()) {
		t.Error("dialog cancellation was not propagated")
	}
}

func testVideo(t *testing.T) {
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
	mod, err := run(`
import pygame as pg
from pygame._sdl2 import Window, Renderer, Texture, Image, get_drivers
assert list(get_drivers())[0].name == 'ebitengine'
w = Window('unit', size=(128, 128), resizable=True)
r = Renderer(w)
r.draw_color = (20, 30, 40, 255)
r.clear()
r.draw_color = (255, 255, 255, 255)
r.draw_line((0, 0), (64, 64))
r.draw_point((72, 32))
r.draw_rect(pg.Rect(0, 64, 64, 32))
r.fill_rect(pg.Rect(0, 96, 64, 32))
snap = r.to_surface()
assert snap.get_at((72, 32)) == pg.Color('white')
assert snap.get_at((16, 16)) == pg.Color('white')
assert snap.get_at((0, 70)) == pg.Color('white')
assert snap.get_at((20, 70)) == (20, 30, 40, 255)
assert snap.get_at((10, 110)) == pg.Color('white')
surf = pg.Surface((8, 8))
surf.fill('red')
changed = surf.fill('green', (4, 0, 4, 4))
assert changed.topleft == (4, 0) and changed.size == (4, 4)
assert surf.get_at((3, 1)) == pg.Color('red')
assert surf.get_at((4, 1)) == pg.Color('green')
tex = Texture(r, (8, 8), streaming=True)
tex.update(surf)
r.draw_color = (0, 0, 255, 255)
r.clear()
tex.draw(dstrect=(20, 10, 16, 16))
Image(tex).draw(dstrect=(50, 20))
tex.draw(srcrect=(4, 0, 4, 4), dstrect=(70, 20, 8, 8))
surf.fill('white')
tex.update(surf)
read = r.to_surface()
assert read.get_at((21, 11)) == pg.Color('red')
assert read.get_at((29, 11)) == pg.Color('green')
assert read.get_at((50, 20)) == pg.Color('red')
assert read.get_at((58, 20)) == pg.Color('blue')
assert read.get_at((72, 22)) == pg.Color('green')
assert snap.get_at((72, 32)) == pg.Color('white')
w2 = Window('second', size=(64, 64), always_on_top=True)
w2.opacity = 0.5
assert w2.opacity == 0.5
r2 = Renderer(w2)
tex2 = Texture.from_surface(r2, surf)
tex2.draw()
assert r2.to_surface().get_at((63, 63)) == pg.Color('white')
w2.destroy()
w2.destroy()
try:
    tex2.draw()
except pg.error:
    pass
else:
    assert False
try:
    tex.update(pg.Surface((2, 2)))
except ValueError:
    pass
else:
    assert False
w.size = (160, 120)
assert r.to_surface().get_size() == (160, 120)
pg.quit()
`, "sdl2 assertions", "video_units")
	if err != nil {
		py.TracebackDump(err)
		t.Fatal(err)
	}
	_ = mod
	script, err := scripts.Load("video.py")
	if err != nil {
		t.Fatal(err)
	}
	// Isolate the full demo from the unit windows above and inject deterministic
	// events/time, not a modified copy of the demo's render loop.
	ctx2 := py.NewContext(py.DefaultContextOpts())
	defer ctx2.Close()
	system2 := &testSystem{}
	if err := RegisterPygameModule(system2, ctx2); err != nil {
		t.Fatal(err)
	}
	execute := func(src, filename string, module interface{}) (*py.Module, error) {
		code, err := py.Compile(src, filename, "exec", 0, false)
		if err != nil {
			return nil, err
		}
		return py.RunCode(ctx2, code, filename, module)
	}
	path := filepath.Join(t.TempDir(), "video.png")
	// Use a Python string constant produced by the compiler's existing repr helper.
	quoted, _ := py.Repr(py.String(path))
	prelude := `
import pygame as pg
import pygame._sdl2 as sdl
import random
sdl.messagebox = lambda *args, **kwargs: 0
frames = 0
pg._ticks = lambda: frames * 100
old_save = pg.image.save
pg.image.save = lambda surface, filename: old_save(surface, SCREENSHOT)
stream_updates = 0
old_update = sdl.Texture.update
def update(self, surface):
    global stream_updates
    old_update(self, surface)
    if self.width == 64 and self.height == 64: stream_updates += 1
sdl.Texture.update = update
board_pixels = []
old_present = sdl.Renderer.present
def present(self):
    old_present(self)
    if self.window is win: board_pixels.append(self.to_surface().get_at((65, 129)))
sdl.Renderer.present = present
old_quit = pg.quit
final_title = ''
def quit():
    global final_title
    final_title = win.title
    old_quit()
pg.quit = quit
class Event:
    def __init__(self, kind, key=None, window=None): self.type, self.key, self.window = kind, key, window
def events():
    global frames
    frames += 1
    if frames == 7: return [Event(pg.KEYDOWN, pg.K_ESCAPE, win2)]
    # String-key-only native dicts are not used for these numeric key codes.
    sequence = (pg.K_RIGHT, pg.K_DOWN, pg.K_SPACE, pg.K_f, pg.K_f, pg.K_s, None, pg.K_LEFT, pg.K_UP, pg.K_SPACE, None, pg.K_ESCAPE)
    key = sequence[frames-1]
    return [] if key is None else [Event(pg.KEYDOWN, key, win)]
pg.event.get = events
random.seed(29)
first_color = tuple([random.randint(0,255) for i in range(3)]) + (255,)
for i in range(255*3): random.randint(0,255)
second_color = tuple([random.randint(0,255) for i in range(3)]) + (255,)
random.seed(29)
`
	prelude = "SCREENSHOT = " + string(quoted.(py.String)) + "\n" + prelude
	mod, err = execute(prelude, "python/video.py", "video_demo")
	if err != nil {
		py.TracebackDump(err)
		t.Fatal(err)
	}
	if _, err = execute(string(script), "python/video.py", mod); err != nil {
		py.TracebackDump(err)
		t.Fatal(err)
	}
	_, err = execute(`
assert frames == 12, 'event count'
assert x == 250 and y == 50 and bg_index == 2 and full == 0, 'controls'
assert stream_updates == 2, 'texture update count'
assert len(board_pixels) == 12, 'present count'
assert board_pixels[0] == first_color and board_pixels[9] == first_color, 'first texture colors'
assert board_pixels[10] == second_color and board_pixels[11] == second_color, 'second texture colors'
assert clock.get_fps() == 10.0, 'fps'
assert final_title[:5] == 'FPS: ', 'title'
`, "video final assertions", mod)
	if err != nil {
		t.Logf("first=%s second=%s pixels=%s", py.DebugRepr(mod.Globals["first_color"]), py.DebugRepr(mod.Globals["second_color"]), py.DebugRepr(mod.Globals["board_pixels"]))
		py.TracebackDump(err)
		t.Fatal(err)
	}
	if system2.frames != 13 {
		t.Errorf("got %d presents, want 13", system2.frames)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	screenshot, err := png.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	if screenshot.Bounds().Size() != image.Pt(640, 480) {
		t.Errorf("screenshot size = %v", screenshot.Bounds())
	}
	if c := color.NRGBAModel.Convert(screenshot.At(639, 479)).(color.NRGBA); c != (color.NRGBA{0, 255, 0, 255}) {
		t.Errorf("screenshot background = %v", c)
	}
	// Choosing No must exit, rather than having SystemExit swallowed by the
	// example's old bare except handler and opening windows anyway.
	ctx3 := py.NewContext(py.DefaultContextOpts())
	defer ctx3.Close()
	system3 := &testSystem{}
	if err := RegisterPygameModule(system3, ctx3); err != nil {
		t.Fatal(err)
	}
	setup, err := py.Compile("import pygame as pg\nimport pygame._sdl2 as sdl\nsdl.messagebox = lambda *args, **kwargs: 1\npg._events = lambda: [(pg.QUIT, 0, None)]\n", "python/video.py", "exec", 0, false)
	if err != nil {
		t.Fatal(err)
	}
	declined, err := py.RunCode(ctx3, setup, "python/video.py", "video_declined")
	if err != nil {
		t.Fatal(err)
	}
	full, err := py.Compile(string(script), "python/video.py", "exec", 0, false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = py.RunCode(ctx3, full, "python/video.py", declined)
	if !py.IsException(py.SystemExit, err) {
		t.Errorf("No response returned %v, want SystemExit", err)
	}
	if system3.frames != 0 {
		t.Errorf("declined dialog rendered %d frames", system3.frames)
	}
}
