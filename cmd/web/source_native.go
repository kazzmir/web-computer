//go:build !js

package main

import "github.com/kazzmir/web-computer/scripts"

func loadPythonSource(path string) ([]byte, error) {
	return scripts.Load(path)
}
