package main

import (
	"syscall/js"

	"github.com/kazzmir/web-computer/scripts"
)

// The browser runner sets this before starting Go. Treat it only as Python
// source, never as HTML/JavaScript or as a filename supplied to the filesystem.
func loadPythonSource(path string) ([]byte, error) {
	source := js.Global().Get("webComputerSource")
	if source.Type() == js.TypeString {
		return []byte(source.String()), nil
	}
	return scripts.Load(path)
}
