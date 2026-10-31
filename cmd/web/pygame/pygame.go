package pygame

import (
	_ "embed"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/kazzmir/web-computer/gpython/py"
)

//go:embed pygame.py
var source string

type GraphicsSystem interface {
	Flip()
	AddDrawCall(func(*ebiten.Image))
	SetScreenSize(int, int)
}

type pixels struct{ image *ebiten.Image }

var pixelsType = py.NewType("_Pixels", "Internal surface storage")

func (p *pixels) Type() *py.Type { return pixelsType }

func rgba(obj py.Object) (color.RGBA, error) {
	values, err := py.SequenceTuple(obj)
	if err != nil {
		return color.RGBA{}, err
	}
	if len(values) != 3 && len(values) != 4 {
		return color.RGBA{}, py.ExceptionNewf(py.ValueError, "expected RGB or RGBA color")
	}
	c := []uint8{0, 0, 0, 255}
	for i, v := range values {
		n, ok := v.(py.Int)
		if !ok || n < 0 || n > 255 {
			return color.RGBA{}, py.ExceptionNewf(py.ValueError, "invalid color component")
		}
		c[i] = uint8(n)
	}
	return color.RGBA{c[0], c[1], c[2], c[3]}, nil
}

func RegisterPygameModule(system GraphicsSystem, context py.Context) error {
	start := time.Now()
	methods, closeAudio := mediaMethods()
	// Decode native arguments before entering a pixel operation, so malformed
	// Python calls raise TypeError rather than panicking on Go assertions.
	add := func(name string, signature string, fn func(py.Tuple) (py.Object, error)) {
		methods = append(methods, py.MustNewMethod(name, func(self py.Object, args py.Tuple) (py.Object, error) {
			if len(args) != len(signature) {
				return nil, py.ExceptionNewf(py.TypeError, "%s expects %d arguments", name, len(signature))
			}
			for i, kind := range signature {
				valid := false
				switch kind {
				case 'p':
					_, valid = args[i].(*pixels)
				case 'i':
					_, valid = args[i].(py.Int)
				case 's':
					_, valid = args[i].(py.String)
				case 'b':
					_, valid = args[i].(py.Bool)
				case 'n':
					switch args[i].(type) {
					case py.Int, py.Float, py.Bool:
						valid = true
					}
				case 'o':
					valid = true
				}
				if !valid {
					return nil, py.ExceptionNewf(py.TypeError, "invalid argument %d to %s", i+1, name)
				}
			}
			return fn(args)
		}, 0, ""))
	}
	integer := func(a py.Object) int { return int(a.(py.Int)) }
	img := func(a py.Object) *ebiten.Image { return a.(*pixels).image }
	add("_new_surface", "ii", func(a py.Tuple) (py.Object, error) {
		w, h := integer(a[0]), integer(a[1])
		if w <= 0 || h <= 0 {
			return nil, py.ExceptionNewf(py.ValueError, "surface dimensions must be positive")
		}
		return &pixels{ebiten.NewImage(w, h)}, nil
	})
	add("_size", "p", func(a py.Tuple) (py.Object, error) {
		b := img(a[0]).Bounds()
		return py.Tuple{py.Int(b.Dx()), py.Int(b.Dy())}, nil
	})
	add("_fill", "po", func(a py.Tuple) (py.Object, error) {
		c, e := rgba(a[1])
		if e != nil {
			return nil, e
		}
		img(a[0]).Fill(c)
		return py.None, nil
	})
	add("_blit", "ppii", func(a py.Tuple) (py.Object, error) {
		var o ebiten.DrawImageOptions
		o.GeoM.Translate(float64(integer(a[2])), float64(integer(a[3])))
		img(a[0]).DrawImage(img(a[1]), &o)
		return py.None, nil
	})
	add("_get_at", "pii", func(a py.Tuple) (py.Object, error) {
		x, y := integer(a[1]), integer(a[2])
		if !image.Pt(x, y).In(img(a[0]).Bounds()) {
			return nil, py.ExceptionNewf(py.IndexError, "pixel outside surface")
		}
		c := color.NRGBAModel.Convert(img(a[0]).At(x, y)).(color.NRGBA)
		return py.Tuple{py.Int(c.R), py.Int(c.G), py.Int(c.B), py.Int(c.A)}, nil
	})
	add("_colorkey", "po", func(a py.Tuple) (py.Object, error) {
		c, e := rgba(a[1])
		if e != nil {
			return nil, e
		}
		im := img(a[0])
		b := im.Bounds()
		data := make([]byte, 4*b.Dx()*b.Dy())
		im.ReadPixels(data)
		for i := 0; i < len(data); i += 4 {
			if data[i] == c.R && data[i+1] == c.G && data[i+2] == c.B {
				data[i] = 0
				data[i+1] = 0
				data[i+2] = 0
				data[i+3] = 0
			}
		}
		im.WritePixels(data)
		return py.None, nil
	})
	add("_load", "s", func(a py.Tuple) (py.Object, error) {
		name := string(a[0].(py.String))
		f, e := openAsset(name)
		if e != nil {
			return nil, py.ExceptionNewf(py.OSError, "loading %s: %v", name, e)
		}
		defer f.Close()
		decoded, _, e := image.Decode(f)
		if e != nil {
			return nil, py.ExceptionNewf(py.OSError, "loading %s: %v", name, e)
		}
		return &pixels{ebiten.NewImageFromImage(decoded)}, nil
	})
	add("_scale", "pii", func(a py.Tuple) (py.Object, error) {
		w, h := integer(a[1]), integer(a[2])
		if w <= 0 || h <= 0 {
			return nil, py.ExceptionNewf(py.ValueError, "scale dimensions must be positive")
		}
		from := img(a[0])
		b := from.Bounds()
		out := ebiten.NewImage(w, h)
		var o ebiten.DrawImageOptions
		o.GeoM.Scale(float64(w)/float64(b.Dx()), float64(h)/float64(b.Dy()))
		out.DrawImage(from, &o)
		return &pixels{out}, nil
	})
	add("_flip", "pbb", func(a py.Tuple) (py.Object, error) {
		from := img(a[0])
		b := from.Bounds()
		out := ebiten.NewImage(b.Dx(), b.Dy())
		var o ebiten.DrawImageOptions
		sx, sy := 1.0, 1.0
		tx, ty := 0.0, 0.0
		if bool(a[1].(py.Bool)) {
			sx = -1
			tx = float64(b.Dx())
		}
		if bool(a[2].(py.Bool)) {
			sy = -1
			ty = float64(b.Dy())
		}
		o.GeoM.Scale(sx, sy)
		o.GeoM.Translate(tx, ty)
		out.DrawImage(from, &o)
		return &pixels{out}, nil
	})
	add("_rotate", "pn", func(a py.Tuple) (py.Object, error) {
		angle := 0.0
		switch v := a[1].(type) {
		case py.Int:
			angle = float64(v)
		case py.Float:
			angle = float64(v)
		case py.Bool:
			if bool(v) {
				angle = 1
			}
		}
		rad := -angle * math.Pi / 180
		from := img(a[0])
		b := from.Bounds()
		w, h := float64(b.Dx()), float64(b.Dy())
		nw := int(math.Ceil(math.Abs(w*math.Cos(rad)) + math.Abs(h*math.Sin(rad)) - 1e-9))
		nh := int(math.Ceil(math.Abs(w*math.Sin(rad)) + math.Abs(h*math.Cos(rad)) - 1e-9))
		out := ebiten.NewImage(nw, nh)
		var o ebiten.DrawImageOptions
		o.GeoM.Translate(-w/2, -h/2)
		o.GeoM.Rotate(rad)
		o.GeoM.Translate(float64(nw)/2, float64(nh)/2)
		out.DrawImage(from, &o)
		return &pixels{out}, nil
	})
	add("_set_mode", "ii", func(a py.Tuple) (py.Object, error) {
		system.SetScreenSize(integer(a[0]), integer(a[1]))
		return py.None, nil
	})
	add("_caption", "s", func(a py.Tuple) (py.Object, error) {
		ebiten.SetWindowTitle(string(a[0].(py.String)))
		return py.None, nil
	})
	add("_present", "p", func(a py.Tuple) (py.Object, error) {
		im := img(a[0])
		system.AddDrawCall(func(screen *ebiten.Image) { screen.DrawImage(im, nil) })
		system.Flip()
		return py.None, nil
	})
	add("_mouse_pos", "", func(a py.Tuple) (py.Object, error) {
		x, y := ebiten.CursorPosition()
		return py.Tuple{py.Int(x), py.Int(y)}, nil
	})
	add("_mouse_visible", "b", func(a py.Tuple) (py.Object, error) {
		was := ebiten.CursorMode() != ebiten.CursorModeHidden
		if bool(a[0].(py.Bool)) {
			ebiten.SetCursorMode(ebiten.CursorModeVisible)
		} else {
			ebiten.SetCursorMode(ebiten.CursorModeHidden)
		}
		return py.Bool(was), nil
	})
	add("_ticks", "", func(a py.Tuple) (py.Object, error) { return py.Int(time.Since(start) / time.Millisecond), nil })
	add("_tps", "i", func(a py.Tuple) (py.Object, error) {
		fps := integer(a[0])
		if fps > 0 {
			ebiten.SetTPS(fps)
		}
		return py.None, nil
	})
	add("_events", "", func(a py.Tuple) (py.Object, error) {
		events := py.Tuple{}
		for _, key := range inpututil.AppendJustPressedKeys(nil) {
			events = append(events, py.Tuple{py.Int(768), py.Int(key)})
		}
		for _, button := range []ebiten.MouseButton{ebiten.MouseButtonLeft, ebiten.MouseButtonMiddle, ebiten.MouseButtonRight} {
			n := py.Int(1)
			if button == ebiten.MouseButtonMiddle {
				n = 2
			}
			if button == ebiten.MouseButtonRight {
				n = 3
			}
			if inpututil.IsMouseButtonJustPressed(button) {
				events = append(events, py.Tuple{py.Int(1025), n})
			}
			if inpututil.IsMouseButtonJustReleased(button) {
				events = append(events, py.Tuple{py.Int(1026), n})
			}
		}
		if ebiten.IsWindowBeingClosed() {
			events = append(events, py.Tuple{py.Int(256), py.Int(0)})
		}
		return events, nil
	})
	globals := py.StringDict{}
	for name, key := range map[string]ebiten.Key{"K_ESCAPE": ebiten.KeyEscape, "K_w": ebiten.KeyW, "K_a": ebiten.KeyA, "K_s": ebiten.KeyS, "K_d": ebiten.KeyD, "K_UP": ebiten.KeyUp, "K_DOWN": ebiten.KeyDown, "K_LEFT": ebiten.KeyLeft, "K_RIGHT": ebiten.KeyRight} {
		globals[name] = py.Int(key)
	}
	py.RegisterModule(&py.ModuleImpl{Info: py.ModuleInfo{Name: "pygame", Doc: "Pygame backed by Ebitengine"}, Methods: methods, Globals: globals, CodeSrc: source,
		OnContextClosed: func(*py.Module) { closeAudio() },
	})
	return nil
}
