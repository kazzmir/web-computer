package main

import (
	"testing"

	"github.com/kazzmir/web-computer/gpython/py"
)

func TestStepOutAvailability(t *testing.T) {
	for _, test := range []struct {
		flags int32
		depth int
		want  bool
	}{
		{0, 1, false}, {0, 2, false}, // module/class frames
		{py.CO_NEWLOCALS | py.CO_OPTIMIZED, 2, true},
		{py.CO_NEWLOCALS | py.CO_GENERATOR, 3, true},
		{py.CO_NEWLOCALS, 1, false},
	} {
		frame := &py.Frame{Code: &py.Code{Flags: test.flags}}
		if got := canStepOut(frame, test.depth); got != test.want {
			t.Errorf("flags=%d depth=%d: got %v", test.flags, test.depth, got)
		}
	}
}

func TestStepOutPauseDepth(t *testing.T) {
	for _, test := range []struct {
		depth            int
		breakpoint, want bool
	}{
		{4, false, false}, {3, false, false}, {2, false, true}, {1, false, true},
		{4, true, true}, {3, true, true},
	} {
		if got := shouldPauseDebug("out", 3, test.depth, test.breakpoint); got != test.want {
			t.Errorf("depth=%d breakpoint=%v: got %v", test.depth, test.breakpoint, got)
		}
	}
}
