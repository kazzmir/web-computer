package pygame

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"path/filepath"
	"strings"

	"github.com/kazzmir/web-computer/gpython/py"
	"golang.org/x/image/bmp"
)

func registerImageSave(add methodAdder) {
	add("_save", "ps", func(args py.Tuple) (py.Object, error) {
		p := args[0].(*pixels)
		if p.image == nil {
			return nil, py.ExceptionNewf(py.ValueError, "cannot save an empty surface")
		}
		filename := string(args[1].(py.String))
		var encode func(io.Writer, image.Image) error
		switch strings.ToLower(filepath.Ext(filename)) {
		case ".png":
			encode = png.Encode
		case ".jpg", ".jpeg":
			encode = func(w io.Writer, img image.Image) error { return jpeg.Encode(w, img, nil) }
		case ".bmp":
			encode = bmp.Encode
		default:
			return nil, py.ExceptionNewf(py.ValueError, "unsupported image format: %s", filename)
		}
		snapshot := image.NewRGBA(p.bounds())
		p.image.ReadPixels(snapshot.Pix)
		var encoded bytes.Buffer
		if err := encode(&encoded, snapshot); err != nil {
			return nil, py.ExceptionNewf(py.OSError, "saving %s: %v", filename, err)
		}
		if err := saveImageData(filename, encoded.Bytes()); err != nil {
			return nil, py.ExceptionNewf(py.OSError, "%v", err)
		}
		return py.None, nil
	})
}
