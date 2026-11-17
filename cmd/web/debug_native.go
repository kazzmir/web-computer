//go:build !js

package main

import "github.com/kazzmir/web-computer/gpython/py"

func setupDebugger(ctx py.Context, filename string, engine *Engine) func() { return func() {} }
