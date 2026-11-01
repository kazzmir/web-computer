package pygame

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/kazzmir/web-computer/gpython/py"
	"golang.org/x/image/font/gofont/goregular"
)

//go:embed sdl2.py
var sdl2Source string

const windowCloseEvent = 32787
const sdlBorder, sdlTitle = 2, 26

type sdlWindow struct {
	id                                  int
	owner                               py.Object
	title                               string
	position                            image.Point
	surface                             *pixels
	panel, icon                         *ebiten.Image
	opacity                             float64
	resizable, top, destroyed, renderer bool
	color                               color.RGBA
}

func (w *sdlWindow) bounds() image.Rectangle {
	return image.Rectangle{Min: w.position, Max: w.position.Add(w.surface.bounds().Size()).Add(image.Pt(2*sdlBorder, sdlTitle+2*sdlBorder))}
}
func (w *sdlWindow) closeRect() image.Rectangle {
	b := w.bounds()
	return image.Rect(b.Max.X-24, b.Min.Y+5, b.Max.X-6, b.Min.Y+23)
}

type sdlTexture struct {
	id        int
	window    *sdlWindow
	pixels    *pixels
	streaming bool
}
type sdlInput struct {
	mouse         image.Point
	pressed, down bool
	keys          []ebiten.Key
}

type sdlDesktop struct {
	system                               GraphicsSystem
	yield                                func() error
	windows                              []*sdlWindow
	textures                             map[int]*sdlTexture
	primary, focused, dragging, resizing *sdlWindow
	dragOffset                           image.Point
	nextID, width, height                int
	scene                                *ebiten.Image
	font                                 *text.GoTextFaceSource
	cancelled                            bool
	input                                func() sdlInput
}

func newSDLDesktop(system GraphicsSystem, yield func() error) *sdlDesktop {
	return &sdlDesktop{system: system, yield: yield, nextID: 1, textures: map[int]*sdlTexture{}, input: func() sdlInput {
		x, y := ebiten.CursorPosition()
		return sdlInput{image.Pt(x, y), inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft), ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft), inpututil.AppendJustPressedKeys(nil)}
	}}
}
func (d *sdlDesktop) check() error {
	if d.cancelled {
		return py.ExceptionNewf(py.SystemExit, "window closed")
	}
	return nil
}
func (d *sdlDesktop) window(id int) (*sdlWindow, error) {
	if err := d.check(); err != nil {
		return nil, err
	}
	for _, w := range d.windows {
		if w.id == id && !w.destroyed {
			return w, nil
		}
	}
	return nil, py.ExceptionNewf(py.OSError, "window has been destroyed or is unknown")
}
func (d *sdlDesktop) renderer(id int) (*sdlWindow, error) {
	w, e := d.window(id)
	if e != nil {
		return nil, e
	}
	if !w.renderer {
		return nil, py.ExceptionNewf(py.OSError, "window has no renderer")
	}
	return w, nil
}
func (d *sdlDesktop) texture(id int) (*sdlTexture, error) {
	if err := d.check(); err != nil {
		return nil, err
	}
	t, ok := d.textures[id]
	if !ok || t.window.destroyed {
		return nil, py.ExceptionNewf(py.OSError, "texture's renderer is no longer available")
	}
	return t, nil
}
func (d *sdlDesktop) active() bool {
	for _, w := range d.windows {
		if !w.destroyed {
			return true
		}
	}
	return false
}
func (d *sdlDesktop) owner() py.Object {
	if d.focused != nil && !d.focused.destroyed {
		return d.focused.owner
	}
	return py.None
}
func (d *sdlDesktop) ordered() []*sdlWindow {
	windows := []*sdlWindow{}
	for _, w := range d.windows {
		if !w.destroyed {
			windows = append(windows, w)
		}
	}
	sort.SliceStable(windows, func(i, j int) bool { return !windows[i].top && windows[j].top })
	return windows
}
func (d *sdlDesktop) focus(w *sdlWindow) {
	d.focused = w
	for i, old := range d.windows {
		if old == w {
			copy(d.windows[i:], d.windows[i+1:])
			d.windows[len(d.windows)-1] = w
			break
		}
	}
}
func (d *sdlDesktop) layout(width, height int) {
	limit := ebiten.MaxImageSize()
	if limit == 0 {
		limit = 4096
	}
	limit = min(limit, 4096)
	width, height = min(width, limit), min(height, limit)
	if width == d.width && height == d.height {
		return
	}
	d.width, d.height = width, height
	d.system.SetScreenSize(width, height)
	ebiten.SetWindowSize(width, height)
	if d.scene != nil {
		d.scene.Deallocate()
	}
	d.scene = ebiten.NewImage(width, height)
}
func (d *sdlDesktop) windowLayout() {
	width, height := 320, 240
	for _, w := range d.windows {
		if !w.destroyed {
			b := w.bounds()
			width = max(width, b.Max.X+12)
			height = max(height, b.Max.Y+12)
		}
	}
	d.layout(width, height)
}
func (d *sdlDesktop) resize(w *sdlWindow, size image.Point) {
	old := w.surface
	w.surface = newPixels(size.X, size.Y)
	w.surface.image.Fill(color.Black)
	blitPixels(w.surface, old, image.Point{}, old.bounds())
	old.image.Deallocate()
	if w.panel != nil {
		w.panel.Deallocate()
		w.panel = nil
	}
}
func (d *sdlDesktop) destroy(w *sdlWindow) {
	w.destroyed = true
	if d.dragging == w {
		d.dragging = nil
	}
	if d.resizing == w {
		d.resizing = nil
	}
	if d.focused == w {
		d.focused = nil
		for _, other := range d.windows {
			if !other.destroyed {
				d.focused = other
				break
			}
		}
	}
	d.windowLayout()
}

func (d *sdlDesktop) poll() py.Tuple {
	events := py.Tuple{}
	if !d.active() {
		return events
	}
	in := d.input()
	if !in.down {
		d.dragging, d.resizing = nil, nil
	}
	if in.pressed {
		d.dragging, d.resizing = nil, nil
		ordered := d.ordered()
		for i := len(ordered) - 1; i >= 0; i-- {
			w := ordered[i]
			if !in.mouse.In(w.bounds()) {
				continue
			}
			d.focus(w)
			if in.mouse.In(w.closeRect()) {
				kind := windowCloseEvent
				if w == d.primary {
					kind = 256
				}
				events = append(events, py.Tuple{py.Int(kind), py.Int(0), w.owner})
			} else if w.resizable && in.mouse.X >= w.bounds().Max.X-12 && in.mouse.Y >= w.bounds().Max.Y-12 {
				d.resizing = w
			} else if in.mouse.Y < w.position.Y+sdlTitle {
				d.dragging = w
				d.dragOffset = in.mouse.Sub(w.position)
			}
			break
		}
	}
	if in.down && d.dragging != nil {
		pos := in.mouse.Sub(d.dragOffset)
		pos.X = max(0, min(pos.X, 1600))
		pos.Y = max(0, min(pos.Y, 1000))
		d.dragging.position = pos
		d.windowLayout()
	}
	if in.down && d.resizing != nil {
		size := in.mouse.Sub(d.resizing.position).Sub(image.Pt(sdlBorder, sdlTitle+sdlBorder))
		size.X = max(64, min(size.X, 1920))
		size.Y = max(64, min(size.Y, 1080))
		if size != d.resizing.surface.bounds().Size() {
			d.resize(d.resizing, size)
			d.windowLayout()
		}
	}
	return events
}
func (d *sdlDesktop) label(dst *ebiten.Image, message string, x, y float64, size float64, clr color.Color) error {
	if d.font == nil {
		source, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
		if err != nil {
			return err
		}
		d.font = source
	}
	options := &text.DrawOptions{}
	options.GeoM.Translate(x, y)
	options.ColorScale.ScaleWithColor(clr)
	text.Draw(dst, message, &text.GoTextFace{Source: d.font, Size: size}, options)
	return nil
}
func (d *sdlDesktop) compose() error {
	d.windowLayout()
	d.scene.Fill(color.RGBA{32, 36, 44, 255})
	for _, w := range d.ordered() {
		size := w.bounds().Size()
		if w.panel == nil {
			w.panel = ebiten.NewImage(size.X, size.Y)
		}
		chrome := color.RGBA{70, 76, 90, 255}
		if w == d.focused {
			chrome = color.RGBA{45, 85, 140, 255}
		}
		w.panel.Fill(chrome)
		var opts ebiten.DrawImageOptions
		opts.GeoM.Translate(sdlBorder, sdlTitle+sdlBorder)
		w.panel.DrawImage(w.surface.image, &opts)
		if w.resizable {
			for offset := float32(4); offset <= 10; offset += 3 {
				vector.StrokeLine(w.panel, float32(size.X)-offset, float32(size.Y)-3, float32(size.X)-3, float32(size.Y)-offset, 1, color.RGBA{180, 180, 185, 255}, false)
			}
		}
		labelX := 8.0
		if w.icon != nil {
			var iconOpts ebiten.DrawImageOptions
			b := w.icon.Bounds()
			iconOpts.GeoM.Scale(18/float64(b.Dx()), 18/float64(b.Dy()))
			iconOpts.GeoM.Translate(5, 4)
			w.panel.DrawImage(w.icon, &iconOpts)
			labelX = 28
		}
		if err := d.label(w.panel, w.title, labelX, 3, 16, color.White); err != nil {
			return err
		}
		vector.FillRect(w.panel, float32(size.X-24), 5, 18, 18, color.RGBA{160, 45, 50, 255}, false)
		vector.StrokeLine(w.panel, float32(size.X-20), 9, float32(size.X-10), 19, 1, color.White, false)
		vector.StrokeLine(w.panel, float32(size.X-10), 9, float32(size.X-20), 19, 1, color.White, false)
		var panelOpts ebiten.DrawImageOptions
		panelOpts.GeoM.Translate(float64(w.position.X), float64(w.position.Y))
		panelOpts.ColorScale.ScaleAlpha(float32(w.opacity))
		d.scene.DrawImage(w.panel, &panelOpts)
	}
	return nil
}
func (d *sdlDesktop) present() error {
	if err := d.check(); err != nil {
		return err
	}
	if err := d.compose(); err != nil {
		return err
	}
	return d.presentScene()
}
func (d *sdlDesktop) presentScene() error {
	scene := d.scene
	d.system.AddDrawCall(func(screen *ebiten.Image) { screen.DrawImage(scene, nil) })
	if err := d.yield(); err != nil {
		d.cancelled = true
		return err
	}
	return nil
}
func (d *sdlDesktop) close() {
	for _, t := range d.textures {
		t.pixels.image.Deallocate()
	}
	for _, w := range d.windows {
		w.destroyed = true
		w.surface.image.Deallocate()
		if w.panel != nil {
			w.panel.Deallocate()
		}
		if w.icon != nil {
			w.icon.Deallocate()
		}
	}
	if d.scene != nil {
		d.scene.Deallocate()
		d.scene = nil
	}
	d.windows = nil
	d.textures = map[int]*sdlTexture{}
	d.primary, d.focused, d.dragging, d.resizing = nil, nil, nil, nil
	d.width, d.height = 0, 0
	d.font = nil
}

func (d *sdlDesktop) messagebox(title, message string, buttons []string, defaultButton, escapeButton int) (int, error) {
	if len(buttons) == 0 || len(buttons) > 8 || defaultButton < 0 || defaultButton >= len(buttons) || escapeButton < 0 || escapeButton >= len(buttons) {
		return 0, py.ExceptionNewf(py.ValueError, "invalid messagebox buttons")
	}
	if err := d.check(); err != nil {
		return 0, err
	}
	d.layout(max(640, len(buttons)*100+40), 240)
	ebiten.SetWindowTitle(title)
	selected := defaultButton
	for {
		in := d.input()
		for _, key := range in.keys {
			switch key {
			case ebiten.KeyEnter:
				return selected, nil
			case ebiten.KeyEscape:
				return escapeButton, nil
			case ebiten.KeyRight, ebiten.KeyTab:
				selected = (selected + 1) % len(buttons)
			case ebiten.KeyLeft:
				selected = (selected + len(buttons) - 1) % len(buttons)
			}
		}
		d.scene.Fill(color.RGBA{32, 36, 44, 255})
		if err := d.label(d.scene, title, 24, 24, 22, color.White); err != nil {
			return 0, err
		}
		if err := d.label(d.scene, message, 24, 72, 18, color.White); err != nil {
			return 0, err
		}
		for i, label := range buttons {
			rect := image.Rect(24+i*100, 160, 114+i*100, 200)
			if in.pressed && in.mouse.In(rect) {
				return i, nil
			}
			clr := color.RGBA{65, 72, 85, 255}
			if i == selected {
				clr = color.RGBA{45, 105, 185, 255}
			}
			vector.FillRect(d.scene, float32(rect.Min.X), float32(rect.Min.Y), 90, 40, clr, false)
			if err := d.label(d.scene, label, float64(rect.Min.X+10), 168, 18, color.White); err != nil {
				return 0, err
			}
		}
		if err := d.presentScene(); err != nil {
			return 0, err
		}
	}
}
