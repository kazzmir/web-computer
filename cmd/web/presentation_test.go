package main

import (
	"errors"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kazzmir/web-computer/lib/coroutine"
)

func TestShortProgramPresentsFinalFrame(t *testing.T) {
	engine := &Engine{ScreenWidth: 640, ScreenHeight: 480}
	draws := 0
	engine.Coroutine = coroutine.MakeCoroutine(func(yield coroutine.YieldFunc) error {
		engine.SetScreenSize(320, 240)
		engine.AddDrawCall(func(*ebiten.Image) { draws++ })
		if err := yield(); err != nil {
			return err
		}
		return ebiten.Termination
	})
	defer engine.Stop()
	for i := 0; i < 3; i++ {
		if err := engine.Update(); err != nil {
			t.Fatalf("update before final draw: %v", err)
		}
	}
	oldSize := ebiten.NewImage(640, 480)
	defer oldSize.Deallocate()
	engine.Draw(oldSize)
	if err := engine.Update(); err != nil {
		t.Fatalf("terminated before layout caught up: %v", err)
	}
	finalSize := ebiten.NewImage(320, 240)
	defer finalSize.Deallocate()
	engine.Draw(finalSize)
	if err := engine.Update(); !errors.Is(err, ebiten.Termination) {
		t.Fatalf("final update = %v", err)
	}
	if draws != 2 {
		t.Errorf("draws = %d, want both old and new layouts", draws)
	}
}
