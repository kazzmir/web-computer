package main

import (
	"errors"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kazzmir/web-computer/cmd/web/pygame"
	"github.com/kazzmir/web-computer/gpython/py"
	"github.com/kazzmir/web-computer/lib/coroutine"
	"github.com/kazzmir/web-computer/scripts"
)

func TestFlipPropagatesCancellation(t *testing.T) {
	engine := &Engine{Yield: func() error { return coroutine.CoroutineCancelled }}
	if err := engine.Flip(); !py.IsException(py.SystemExit, err) {
		t.Errorf("Flip = %v, want SystemExit", err)
	}
	failure := errors.New("yield failed")
	engine.Yield = func() error { return failure }
	if err := engine.Flip(); err != failure {
		t.Errorf("Flip = %v, want original yield error", err)
	}
}

// Returning Termination without resuming Python simulates Ebitengine closing
// the native window: the script never gets another Update to consume QUIT.
type closingGame struct {
	*Engine
	updated bool
}

func (game *closingGame) Update() error {
	if game.updated {
		return ebiten.Termination
	}
	game.updated = true
	return game.Engine.Update()
}

func TestWindowCloseStopsPython(t *testing.T) {
	ctx := py.NewContext(py.DefaultContextOpts())
	script, err := scripts.Load("glcube.py")
	if err != nil {
		t.Fatal(err)
	}
	code, err := py.Compile(string(script), "python/glcube.py", "exec", 0, false)
	if err != nil {
		t.Fatal(err)
	}
	mainCode, err := py.Compile("try:\n    main()\nfinally:\n    shutdown_finished = True\n", "shutdown test", "exec", 0, false)
	if err != nil {
		t.Fatal(err)
	}
	engine := &Engine{ScreenWidth: 640, ScreenHeight: 480}
	var module *py.Module
	var scriptErr error
	engine.Coroutine = coroutine.MakeCoroutine(func(yield coroutine.YieldFunc) error {
		engine.Yield = yield
		module, scriptErr = py.RunCode(ctx, code, "python/glcube.py", "glcube_shutdown_test")
		if scriptErr != nil {
			return scriptErr
		}
		_, scriptErr = ctx.RunCode(mainCode, module.Globals, module.Globals, nil)
		return scriptErr
	})
	if err := pygame.RegisterPygameModule(engine, ctx); err != nil {
		engine.Stop()
		ctx.Close()
		t.Fatal(err)
	}
	if err := ebiten.RunGame(&closingGame{Engine: engine}); err != nil {
		t.Errorf("RunGame: %v", err)
	}
	stopped := make(chan error, 1)
	go func() {
		engine.Stop()
		engine.Stop() // Shutdown is idempotent.
		stopped <- ctx.Close()
	}()
	select {
	case err := <-stopped:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("window closed but Python/interpreter shutdown hung")
	}
	if !py.IsException(py.SystemExit, scriptErr) {
		t.Errorf("script returned %v, want SystemExit", scriptErr)
	}
	if module == nil || module.Globals["shutdown_finished"] != py.True {
		t.Error("Python finally block did not execute during shutdown")
	}
}
