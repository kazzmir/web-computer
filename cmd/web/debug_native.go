//go:build !js

package main

import "github.com/kazzmir/web-computer/gpython/py"

func setupDebugger(ctx py.Context, filename string) func() { return func() {} }
