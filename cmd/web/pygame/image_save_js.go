package pygame

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall/js"
)

func saveImageData(filename string, data []byte) error {
	document := js.Global().Get("document")
	if document.IsUndefined() || document.IsNull() {
		return os.WriteFile(filename, data, 0666)
	}
	blobClass, urlAPI := js.Global().Get("Blob"), js.Global().Get("URL")
	if blobClass.Type() != js.TypeFunction || urlAPI.IsUndefined() || urlAPI.Get("createObjectURL").Type() != js.TypeFunction {
		return fmt.Errorf("image downloads are unavailable in this browser")
	}
	mime := "image/png"
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".bmp":
		mime = "image/bmp"
	}
	array := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(array, data)
	blob := blobClass.New([]any{array}, map[string]any{"type": mime})
	url := urlAPI.Call("createObjectURL", blob).String()
	link := document.Call("createElement", "a")
	link.Set("href", url)
	link.Set("download", filepath.Base(filename))
	document.Get("body").Call("appendChild", link)
	link.Call("click")
	link.Call("remove")
	// Use a native JS callback so cleanup still works if the Go game exits.
	revoke := urlAPI.Get("revokeObjectURL").Call("bind", urlAPI)
	js.Global().Call("setTimeout", revoke, 1000, url)
	return nil
}
