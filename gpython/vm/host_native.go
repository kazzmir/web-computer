//go:build !js

package vm

// Native execution needs no browser event-loop handoff; this call is inlined.
func yieldToHost() {}
