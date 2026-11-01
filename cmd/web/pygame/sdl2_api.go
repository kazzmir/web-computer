package pygame

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/kazzmir/web-computer/gpython/py"
)

func sdlPoint(obj py.Object) (image.Point, error) {
	seq, err := py.SequenceTuple(obj)
	if err != nil {
		return image.Point{}, err
	}
	if len(seq) != 2 {
		return image.Point{}, py.ExceptionNewf(py.TypeError, "expected two coordinates")
	}
	values := [2]int{}
	for i, v := range seq {
		n, err := py.FloatAsFloat64(v)
		if err != nil {
			return image.Point{}, err
		}
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return image.Point{}, py.ExceptionNewf(py.ValueError, "coordinates must be finite")
		}
		values[i] = int(n)
	}
	return image.Pt(values[0], values[1]), nil
}
func sdlSize(obj py.Object) (image.Point, error) {
	size, err := sdlPoint(obj)
	if err != nil {
		return size, err
	}
	if size.X < 1 || size.Y < 1 || size.X > 2048 || size.Y > 2048 {
		return size, py.ExceptionNewf(py.ValueError, "dimensions must be between 1 and 2048")
	}
	return size, nil
}
func sdlWindowSize(obj py.Object) (image.Point, error) {
	size, err := sdlSize(obj)
	if err != nil {
		return size, err
	}
	limit := ebiten.MaxImageSize()
	if limit == 0 {
		limit = 2048
	}
	if size.X+2*sdlBorder > limit || size.Y+sdlTitle+2*sdlBorder > limit {
		return size, py.ExceptionNewf(py.ValueError, "window and decorations exceed the graphics size limit")
	}
	return size, nil
}
func pointTuple(p image.Point) py.Tuple { return py.Tuple{py.Int(p.X), py.Int(p.Y)} }
func colorTuple(c color.RGBA) py.Tuple {
	return py.Tuple{py.Int(c.R), py.Int(c.G), py.Int(c.B), py.Int(c.A)}
}
func sdlColor(c color.RGBA) color.NRGBA { return color.NRGBA{R: c.R, G: c.G, B: c.B, A: c.A} }

func (d *sdlDesktop) register(add methodAdder) {
	add("_sdl_quit", "", func(a py.Tuple) (py.Object, error) { d.close(); return py.None, nil })
	add("_sdl_window_new", "osobb", func(a py.Tuple) (py.Object, error) {
		if err := d.check(); err != nil {
			return nil, err
		}
		size, err := sdlWindowSize(a[2])
		if err != nil {
			return nil, err
		}
		x := 12
		for _, w := range d.windows {
			if !w.destroyed {
				x = max(x, w.bounds().Max.X+16)
			}
		}
		w := &sdlWindow{id: d.nextID, owner: a[0], title: string(a[1].(py.String)), position: image.Pt(x, 12), surface: newPixels(size.X, size.Y), opacity: 1, resizable: bool(a[3].(py.Bool)), top: bool(a[4].(py.Bool)), color: color.RGBA{A: 255}}
		d.nextID++
		w.surface.image.Fill(color.Black)
		d.windows = append(d.windows, w)
		if d.primary == nil {
			d.primary = w
			d.focused = w
			ebiten.SetWindowTitle(w.title)
		}
		d.windowLayout()
		return py.Int(w.id), nil
	})
	add("_sdl_window_get", "is", func(a py.Tuple) (py.Object, error) {
		w, e := d.window(int(a[0].(py.Int)))
		if e != nil {
			return nil, e
		}
		switch a[1].(py.String) {
		case "title":
			return py.String(w.title), nil
		case "size":
			return pointTuple(w.surface.bounds().Size()), nil
		case "position":
			return pointTuple(w.position), nil
		case "opacity":
			return py.Float(w.opacity), nil
		}
		return nil, py.ExceptionNewf(py.ValueError, "unknown window property")
	})
	add("_sdl_window_set", "iso", func(a py.Tuple) (py.Object, error) {
		w, e := d.window(int(a[0].(py.Int)))
		if e != nil {
			return nil, e
		}
		switch a[1].(py.String) {
		case "title":
			v, ok := a[2].(py.String)
			if !ok {
				return nil, py.ExceptionNewf(py.TypeError, "title must be a string")
			}
			w.title = string(v)
			if w == d.primary {
				ebiten.SetWindowTitle(w.title)
			}
		case "opacity":
			v, e := py.FloatAsFloat64(a[2])
			if e != nil {
				return nil, e
			}
			if math.IsNaN(v) || v < 0 || v > 1 {
				return nil, py.ExceptionNewf(py.ValueError, "opacity must be between zero and one")
			}
			w.opacity = v
		case "position":
			v, e := sdlPoint(a[2])
			if e != nil {
				return nil, e
			}
			w.position = image.Pt(max(0, min(v.X, 1600)), max(0, min(v.Y, 1000)))
			d.windowLayout()
		case "size":
			v, e := sdlWindowSize(a[2])
			if e != nil {
				return nil, e
			}
			d.resize(w, v)
			d.windowLayout()
		default:
			return nil, py.ExceptionNewf(py.ValueError, "unknown window property")
		}
		return py.None, nil
	})
	add("_sdl_window_icon", "ip", func(a py.Tuple) (py.Object, error) {
		w, e := d.window(int(a[0].(py.Int)))
		if e != nil {
			return nil, e
		}
		src := a[1].(*pixels)
		if src.image == nil {
			return nil, py.ExceptionNewf(py.ValueError, "icon must not be empty")
		}
		if w.icon != nil {
			w.icon.Deallocate()
		}
		w.icon = ebiten.NewImage(src.bounds().Dx(), src.bounds().Dy())
		w.icon.DrawImage(src.image, nil)
		if w == d.primary {
			icon := image.NewRGBA(src.bounds())
			src.image.ReadPixels(icon.Pix)
			ebiten.SetWindowIcon([]image.Image{icon})
		}
		return py.None, nil
	})
	add("_sdl_window_fullscreen", "ib", func(a py.Tuple) (py.Object, error) {
		if _, e := d.window(int(a[0].(py.Int))); e != nil {
			return nil, e
		}
		ebiten.SetFullscreen(bool(a[1].(py.Bool)))
		return py.None, nil
	})
	add("_sdl_window_focus", "i", func(a py.Tuple) (py.Object, error) {
		w, e := d.window(int(a[0].(py.Int)))
		if e != nil {
			return nil, e
		}
		d.focus(w)
		return py.None, nil
	})
	add("_sdl_window_destroy", "i", func(a py.Tuple) (py.Object, error) {
		if err := d.check(); err != nil {
			return nil, err
		}
		for _, w := range d.windows {
			if w.id == int(a[0].(py.Int)) {
				d.destroy(w)
				return py.None, nil
			}
		}
		return nil, py.ExceptionNewf(py.OSError, "unknown window")
	})
	add("_sdl_renderer_new", "i", func(a py.Tuple) (py.Object, error) {
		w, e := d.window(int(a[0].(py.Int)))
		if e != nil {
			return nil, e
		}
		if w.renderer {
			return nil, py.ExceptionNewf(py.OSError, "window already has a renderer")
		}
		w.renderer = true
		return py.Int(w.id), nil
	})
	add("_sdl_renderer_color", "io", func(a py.Tuple) (py.Object, error) {
		w, e := d.renderer(int(a[0].(py.Int)))
		if e != nil {
			return nil, e
		}
		if a[1] != py.None {
			c, e := rgba(a[1])
			if e != nil {
				return nil, e
			}
			w.color = c
		}
		return colorTuple(w.color), nil
	})
	add("_sdl_renderer_shape", "iso", func(a py.Tuple) (py.Object, error) {
		w, e := d.renderer(int(a[0].(py.Int)))
		if e != nil {
			return nil, e
		}
		target := w.surface.image
		c := sdlColor(w.color)
		switch a[1].(py.String) {
		case "clear":
			target.Fill(c)
		case "point":
			pos, e := sdlPoint(a[2])
			if e != nil {
				return nil, e
			}
			target.Set(pos.X, pos.Y, c)
		case "line":
			pair, e := py.SequenceTuple(a[2])
			if e != nil {
				return nil, e
			}
			if len(pair) != 2 {
				return nil, py.ExceptionNewf(py.TypeError, "line requires two points")
			}
			p, e := sdlPoint(pair[0])
			if e != nil {
				return nil, e
			}
			q, e := sdlPoint(pair[1])
			if e != nil {
				return nil, e
			}
			vector.StrokeLine(target, float32(p.X)+0.5, float32(p.Y)+0.5, float32(q.X)+0.5, float32(q.Y)+0.5, 1, c, false)
		case "rect", "fill":
			r, e := surfaceRect(a[2])
			if e != nil {
				return nil, e
			}
			if r.Empty() {
				return py.None, nil
			}
			if a[1] == py.String("fill") {
				if r = r.Intersect(w.surface.bounds()); !r.Empty() {
					target.SubImage(r).(*ebiten.Image).Fill(c)
				}
			} else {
				vector.StrokeRect(target, float32(r.Min.X)+0.5, float32(r.Min.Y)+0.5, float32(r.Dx()-1), float32(r.Dy()-1), 1, c, false)
			}
		default:
			return nil, py.ExceptionNewf(py.ValueError, "unknown drawing operation")
		}
		return py.None, nil
	})
	add("_sdl_present", "i", func(a py.Tuple) (py.Object, error) {
		if _, e := d.renderer(int(a[0].(py.Int))); e != nil {
			return nil, e
		}
		return py.None, d.present()
	})
	add("_sdl_renderer_read", "i", func(a py.Tuple) (py.Object, error) {
		w, e := d.renderer(int(a[0].(py.Int)))
		if e != nil {
			return nil, e
		}
		b := w.surface.bounds()
		out := newPixels(b.Dx(), b.Dy())
		out.image.DrawImage(w.surface.image, nil)
		return out, nil
	})
	add("_sdl_texture_new", "iob", func(a py.Tuple) (py.Object, error) {
		w, e := d.renderer(int(a[0].(py.Int)))
		if e != nil {
			return nil, e
		}
		size, e := sdlSize(a[1])
		if e != nil {
			return nil, e
		}
		id := d.nextID
		d.nextID++
		d.textures[id] = &sdlTexture{id: id, window: w, pixels: newPixels(size.X, size.Y), streaming: bool(a[2].(py.Bool))}
		return py.Int(id), nil
	})
	add("_sdl_texture_size", "i", func(a py.Tuple) (py.Object, error) {
		t, e := d.texture(int(a[0].(py.Int)))
		if e != nil {
			return nil, e
		}
		return pointTuple(t.pixels.bounds().Size()), nil
	})
	add("_sdl_texture_update", "ip", func(a py.Tuple) (py.Object, error) {
		t, e := d.texture(int(a[0].(py.Int)))
		if e != nil {
			return nil, e
		}
		src := a[1].(*pixels)
		if src.bounds().Size() != t.pixels.bounds().Size() {
			return nil, py.ExceptionNewf(py.ValueError, "texture and surface sizes must match")
		}
		scaleInto(src, t.pixels, false)
		return py.None, nil
	})
	add("_sdl_texture_draw", "ioo", func(a py.Tuple) (py.Object, error) {
		t, e := d.texture(int(a[0].(py.Int)))
		if e != nil {
			return nil, e
		}
		src, e := surfaceRect(a[1])
		if e != nil {
			return nil, e
		}
		dst := t.window.surface.bounds()
		if a[2] != py.None {
			dst, e = surfaceRect(a[2])
			if e != nil {
				return nil, e
			}
		}
		if src.Empty() || dst.Empty() {
			return py.None, nil
		}
		visible := src.Intersect(t.pixels.bounds())
		if visible.Empty() {
			return py.None, nil
		}
		sx, sy := float64(dst.Dx())/float64(src.Dx()), float64(dst.Dy())/float64(src.Dy())
		var options ebiten.DrawImageOptions
		options.GeoM.Scale(sx, sy)
		options.GeoM.Translate(float64(dst.Min.X)+float64(visible.Min.X-src.Min.X)*sx, float64(dst.Min.Y)+float64(visible.Min.Y-src.Min.Y)*sy)
		t.window.surface.image.DrawImage(t.pixels.image.SubImage(visible).(*ebiten.Image), &options)
		return py.None, nil
	})
	add("_sdl_messagebox", "ssoii", func(a py.Tuple) (py.Object, error) {
		objects, e := py.SequenceTuple(a[2])
		if e != nil {
			return nil, e
		}
		buttons := make([]string, len(objects))
		for i, obj := range objects {
			label, ok := obj.(py.String)
			if !ok {
				return nil, py.ExceptionNewf(py.TypeError, "button labels must be strings")
			}
			buttons[i] = string(label)
		}
		chosen, e := d.messagebox(string(a[0].(py.String)), string(a[1].(py.String)), buttons, int(a[3].(py.Int)), int(a[4].(py.Int)))
		return py.Int(chosen), e
	})
	py.RegisterModule(&py.ModuleImpl{Info: py.ModuleInfo{Name: "pygame._sdl2", Doc: "SDL2 renderer API with interactive logical windows in an Ebitengine desktop"}, CodeSrc: sdl2Source})
}
