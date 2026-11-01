package pygame

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kazzmir/web-computer/gpython/py"
	"github.com/kazzmir/web-computer/scripts"
	"golang.org/x/image/font/gofont/goregular"
)

func openAsset(name string) (io.ReadCloser, error) {
	if f, err := os.Open(name); err == nil {
		return f, nil
	}
	normalized := filepath.ToSlash(name)
	if i := strings.LastIndex(normalized, "/python/"); i >= 0 {
		normalized = normalized[i+1:]
	}
	if !strings.HasPrefix(normalized, "python/") {
		normalized = "python/" + normalized
	}
	return scripts.Python.Open(normalized)
}

type fontFace struct {
	source *text.GoTextFaceSource
	size   float64
	italic bool
}

var fontType = py.NewType("_Font", "Internal font")

func (*fontFace) Type() *py.Type { return fontType }

type sound struct {
	data    []byte
	players []*audio.Player
}

var soundType = py.NewType("_Sound", "Internal sound")

func (*sound) Type() *py.Type { return soundType }

var audioOnce sync.Once
var audioContext *audio.Context

func getAudioContext() *audio.Context {
	audioOnce.Do(func() { audioContext = audio.NewContext(44100) })
	return audioContext
}

func mediaMethods() ([]*py.Method, func()) {
	music, closeMusic := musicMethods()
	var sounds []*sound
	closeAudio := func() {
		closeMusic()
		for _, s := range sounds {
			for _, p := range s.players {
				p.Close()
			}
			s.players = nil
		}
	}
	return append([]*py.Method{
		py.MustNewMethod("_mixer_init", func(self py.Object, args py.Tuple) (py.Object, error) {
			if err := py.UnpackTuple(args, nil, "get_init", 0, 0); err != nil {
				return nil, err
			}
			if !audioAvailable() {
				return py.None, nil
			}
			return py.Tuple{py.Int(44100), py.Int(-16), py.Int(2)}, nil
		}, 0, ""),
		py.MustNewMethod("_font_italic", func(self py.Object, args py.Tuple) (py.Object, error) {
			var obj, italic py.Object
			if err := py.UnpackTuple(args, nil, "set_italic", 2, 2, &obj, &italic); err != nil {
				return nil, err
			}
			f, ok := obj.(*fontFace)
			if !ok {
				return nil, py.ExceptionNewf(py.TypeError, "expected font")
			}
			enabled, ok := italic.(py.Bool)
			if !ok {
				return nil, py.ExceptionNewf(py.TypeError, "expected bool")
			}
			f.italic = bool(enabled)
			return py.None, nil
		}, 0, ""),
		py.MustNewMethod("_audio_quit", func(self py.Object, args py.Tuple) (py.Object, error) {
			closeAudio()
			return py.None, nil
		}, 0, ""),
		py.MustNewMethod("_font", func(self py.Object, args py.Tuple) (py.Object, error) {
			var filename py.Object
			var sizeArg py.Object
			if err := py.ParseTupleAndKeywords(args, nil, "Oi:Font", []string{"filename", "size"}, &filename, &sizeArg); err != nil {
				return nil, err
			}
			size := sizeArg.(py.Int)
			if size <= 0 {
				return nil, py.ExceptionNewf(py.ValueError, "font size must be positive")
			}
			data := goregular.TTF
			if filename != py.None {
				name, ok := filename.(py.String)
				if !ok {
					return nil, py.ExceptionNewf(py.TypeError, "expected font filename")
				}
				f, e := openAsset(string(name))
				if e != nil {
					return nil, py.ExceptionNewf(py.OSError, "%v", e)
				}
				defer f.Close()
				data, e = io.ReadAll(f)
				if e != nil {
					return nil, py.ExceptionNewf(py.OSError, "%v", e)
				}
			}
			source, e := text.NewGoTextFaceSource(bytes.NewReader(data))
			if e != nil {
				return nil, py.ExceptionNewf(py.ValueError, "%v", e)
			}
			return &fontFace{source: source, size: float64(size)}, nil
		}, 0, ""),
		py.MustNewMethod("_render", func(self py.Object, args py.Tuple) (py.Object, error) {
			var obj py.Object
			var message py.Object
			var antialias py.Object
			var col py.Object
			if err := py.ParseTupleAndKeywords(args, nil, "OsOO:render", []string{"font", "text", "antialias", "color"}, &obj, &message, &antialias, &col); err != nil {
				return nil, err
			}
			f, ok := obj.(*fontFace)
			if !ok {
				return nil, py.ExceptionNewf(py.TypeError, "expected font")
			}
			c, e := rgba(col)
			if e != nil {
				return nil, e
			}
			face := &text.GoTextFace{Source: f.source, Size: f.size}
			width, height := text.Measure(string(message.(py.String)), face, f.size*1.2)
			w, h := int(width+1), int(height+1)
			if w < 1 {
				w = 1
			}
			if h < 1 {
				h = 1
			}
			if f.italic {
				w += int(float64(h)*0.22) + 1
			}
			out := ebiten.NewImage(w, h)
			options := &text.DrawOptions{}
			if f.italic {
				options.GeoM.SetElement(0, 1, -0.22)
				options.GeoM.Translate(float64(h)*0.22, 0)
			}
			options.ColorScale.ScaleWithColor(c)
			options.LineSpacing = f.size * 1.2
			text.Draw(out, string(message.(py.String)), face, options)
			return &pixels{image: out}, nil
		}, 0, ""),
		py.MustNewMethod("_sound", func(self py.Object, args py.Tuple) (py.Object, error) {
			var filename py.Object
			if e := py.ParseTupleAndKeywords(args, nil, "s:Sound", []string{"filename"}, &filename); e != nil {
				return nil, e
			}
			data, e := decodeWAV(string(filename.(py.String)))
			if e != nil {
				return nil, e
			}
			s := &sound{data: data}
			sounds = append(sounds, s)
			return s, nil
		}, 0, ""),
		py.MustNewMethod("_play", func(self py.Object, args py.Tuple) (py.Object, error) {
			var obj py.Object
			if e := py.UnpackTuple(args, nil, "play", 1, 1, &obj); e != nil {
				return nil, e
			}
			if !audioAvailable() {
				return py.None, nil
			}
			s, ok := obj.(*sound)
			if !ok {
				return nil, py.ExceptionNewf(py.TypeError, "expected sound")
			}
			active := s.players[:0]
			for _, p := range s.players {
				if p.IsPlaying() {
					active = append(active, p)
				} else {
					p.Close()
				}
			}
			s.players = active
			player := getAudioContext().NewPlayerFromBytes(s.data)
			s.players = append(s.players, player)
			player.Play()
			return py.None, nil
		}, 0, ""),
	}, music...), closeAudio
}
