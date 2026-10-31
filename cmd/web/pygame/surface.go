package pygame

import (
	_ "golang.org/x/image/bmp"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/kazzmir/web-computer/gpython/py"
)

type nativeFunc = func(py.Tuple) (py.Object, error)
type methodAdder = func(string, string, nativeFunc)

// Surface coordinates are local, even when Ebitengine's SubImage has a
// nonzero bounds origin. Clips are independent on each shared surface view.
func (p *pixels) clipBounds() image.Rectangle {
	if p.clip == nil {
		return p.bounds()
	}
	return p.bounds().Intersect(*p.clip)
}

func (p *pixels) target() *ebiten.Image {
	if p.image == nil || p.clipBounds().Empty() {
		return nil
	}
	return p.image.SubImage(p.clipBounds().Add(p.image.Bounds().Min)).(*ebiten.Image)
}

func rectTuple(r image.Rectangle) py.Tuple {
	return py.Tuple{py.Int(r.Min.X), py.Int(r.Min.Y), py.Int(r.Dx()), py.Int(r.Dy())}
}

func scaleInto(from, to *pixels, smooth bool) {
	target := to.target()
	if from.image == nil || target == nil {
		return
	}
	src, dst := from.bounds(), to.bounds()
	var options ebiten.DrawImageOptions
	if smooth {
		options.Filter = ebiten.FilterLinear
	}
	options.GeoM.Scale(float64(dst.Dx())/float64(src.Dx()), float64(dst.Dy())/float64(src.Dy()))
	origin := to.image.Bounds().Min
	options.GeoM.Translate(float64(origin.X), float64(origin.Y))
	options.Blend = ebiten.BlendCopy
	target.DrawImage(from.image, &options)
}

func registerSurfaceMethods(add methodAdder) {
	add("_get_clip", "p", func(a py.Tuple) (py.Object, error) { return rectTuple(a[0].(*pixels).clipBounds()), nil })
	add("_set_clip", "po", func(a py.Tuple) (py.Object, error) {
		p := a[0].(*pixels)
		if a[1] == py.None {
			p.clip = nil
			return py.None, nil
		}
		coords, e := py.SequenceTuple(a[1])
		if e != nil {
			return nil, e
		}
		if len(coords) != 4 {
			return nil, py.ExceptionNewf(py.TypeError, "expected four rectangle coordinates")
		}
		v := [4]int{}
		for i, coord := range coords {
			n, ok := coord.(py.Int)
			if !ok {
				return nil, py.ExceptionNewf(py.TypeError, "rectangle coordinates must be integers")
			}
			v[i] = int(n)
		}
		r := image.Rectangle{Min: image.Pt(v[0], v[1]), Max: image.Pt(v[0]+v[2], v[1]+v[3])}
		r = r.Intersect(p.bounds())
		p.clip = &r
		return py.None, nil
	})
	add("_subsurface", "piiii", func(a py.Tuple) (py.Object, error) {
		p := a[0].(*pixels)
		x, y, w, h := int(a[1].(py.Int)), int(a[2].(py.Int)), int(a[3].(py.Int)), int(a[4].(py.Int))
		b := p.bounds()
		if x < 0 || y < 0 || w < 0 || h < 0 || x > b.Dx() || y > b.Dy() || w > b.Dx()-x || h > b.Dy()-y {
			return nil, py.ExceptionNewf(py.ValueError, "subsurface rectangle outside surface area")
		}
		if w == 0 || h == 0 {
			return newPixels(w, h), nil
		}
		r := image.Rect(x, y, x+w, y+h).Add(p.image.Bounds().Min)
		return &pixels{image: p.image.SubImage(r).(*ebiten.Image)}, nil
	})
	add("_scroll", "pii", func(a py.Tuple) (py.Object, error) {
		p := a[0].(*pixels)
		clip := p.clipBounds()
		dx, dy := int(a[1].(py.Int)), int(a[2].(py.Int))
		if p.image == nil || clip.Empty() || dx == 0 && dy == 0 {
			return py.None, nil
		}
		// Copy only pixels which remain inside the clip. Uncovered pixels are left
		// unchanged, as in pygame, and the temporary avoids overlapping GPU blits.
		shift := image.Pt(dx, dy)
		dst := clip.Intersect(clip.Add(shift))
		if dst.Empty() {
			return py.None, nil
		}
		origin := p.image.Bounds().Min
		src := dst.Sub(shift).Add(origin)
		snapshot := ebiten.NewImage(dst.Dx(), dst.Dy())
		snapshot.DrawImage(p.image.SubImage(src).(*ebiten.Image), nil)
		defer snapshot.Deallocate()
		var options ebiten.DrawImageOptions
		options.GeoM.Translate(float64(dst.Min.X+origin.X), float64(dst.Min.Y+origin.Y))
		options.Blend = ebiten.BlendCopy
		p.target().DrawImage(snapshot, &options)
		return py.None, nil
	})
	add("_scale_to", "ppiib", func(a py.Tuple) (py.Object, error) {
		from, to := a[0].(*pixels), a[1].(*pixels)
		w, h := int(a[2].(py.Int)), int(a[3].(py.Int))
		if w < 0 || h < 0 || to.bounds().Dx() != w || to.bounds().Dy() != h {
			return nil, py.ExceptionNewf(py.ValueError, "destination surface must match the requested scale size")
		}
		if from.image == to.image && from.image != nil {
			return nil, py.ExceptionNewf(py.ValueError, "source and destination surfaces must differ")
		}
		scaleInto(from, to, bool(a[4].(py.Bool)))
		return py.None, nil
	})
	add("_polygon", "poo", func(a py.Tuple) (py.Object, error) {
		p := a[0].(*pixels)
		c, e := rgba(a[1])
		if e != nil {
			return nil, e
		}
		points, e := py.SequenceTuple(a[2])
		if e != nil {
			return nil, e
		}
		if len(points) < 3 {
			return nil, py.ExceptionNewf(py.ValueError, "polygon requires at least three points")
		}
		var path vector.Path
		var bounds image.Rectangle
		origin := image.Point{}
		if p.image != nil {
			origin = p.image.Bounds().Min
		}
		for i, point := range points {
			pair, e := py.SequenceTuple(point)
			if e != nil {
				return nil, e
			}
			if len(pair) != 2 {
				return nil, py.ExceptionNewf(py.TypeError, "points must have two coordinates")
			}
			coords := [2]int{}
			for j, v := range pair {
				switch n := v.(type) {
				case py.Int:
					coords[j] = int(n)
				case py.Float:
					coords[j] = int(n)
				default:
					return nil, py.ExceptionNewf(py.TypeError, "point coordinates must be numeric")
				}
			}
			pos := image.Pt(coords[0], coords[1])
			r := image.Rectangle{Min: pos, Max: pos.Add(image.Pt(1, 1))}
			if i == 0 {
				bounds = r
				path.MoveTo(float32(pos.X+origin.X), float32(pos.Y+origin.Y))
			} else {
				bounds = bounds.Union(r)
				path.LineTo(float32(pos.X+origin.X), float32(pos.Y+origin.Y))
			}
		}
		path.Close()
		if target := p.target(); target != nil {
			options := &vector.DrawPathOptions{Blend: ebiten.BlendCopy}
			options.ColorScale.ScaleWithColor(c)
			vector.FillPath(target, &path, nil, options)
		}
		return rectTuple(bounds.Intersect(p.clipBounds())), nil
	})
}
