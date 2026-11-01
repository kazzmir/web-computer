package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	// "reflect"

	"github.com/kazzmir/web-computer/cmd/web/pygame"
	"github.com/kazzmir/web-computer/lib/coroutine"
	"github.com/kazzmir/web-computer/scripts"

	_ "github.com/kazzmir/web-computer/gpython/lib"
	"github.com/kazzmir/web-computer/gpython/py"
	_ "github.com/kazzmir/web-computer/gpython/stdlib"

	"github.com/hajimehoshi/ebiten/v2"
	// "github.com/hajimehoshi/ebiten/v2/inpututil"
)

func webtest_foo(self py.Object, args py.Tuple) (py.Object, error) {
	log.Printf("Invoking go foo from python")
	return py.String("foo"), nil
}

func registerWebTestModule() {
	methods := []*py.Method{
		py.MustNewMethod("foo", webtest_foo, 0, ""),
	}

	py.RegisterModule(&py.ModuleImpl{
		Info: py.ModuleInfo{
			Name: "webtest",
			Doc:  "webtest module",
		},
		Methods: methods,
		Globals: py.StringDict{},
		OnContextClosed: func(instance *py.Module) {
		},
	})
}

func test1() {
	log.Printf("Starting python test")
	opts := py.DefaultContextOpts()

	python := py.NewContext(opts)
	defer python.Close()

	script, err := scripts.Load("xmod.py")
	if err != nil {
		log.Printf("Unable to load python script: %v", err)
		return
	}

	registerWebTestModule()

	code, err := py.Compile(string(script), "xmod.py", "exec", 0, false)
	if err != nil {
		log.Printf("Unable to compile python script: %v", err)
		return
	}

	module, err := py.RunCode(python, code, "script", "user")
	if err != nil {
		log.Printf("Unable to run code: %v", err)
	} else {
		log.Printf("Module: %v", module)
	}
}

type Engine struct {
	Coroutine *coroutine.Coroutine
	Yield     coroutine.YieldFunc
	DrawCalls []func(*ebiten.Image)

	ScreenWidth  int
	ScreenHeight int

	finished        bool
	finalFrameDrawn bool
}

// invoked from python when the user calls pygame.display.flip() or update()
func (engine *Engine) Flip() error {
	err := engine.Yield()
	if errors.Is(err, coroutine.CoroutineCancelled) {
		// Unwind Python (including finally blocks) instead of letting it
		// continue drawing and block at another yield after RunGame exits.
		return py.ExceptionNewf(py.SystemExit, "window closed")
	}
	return err
}

// Stop joins the Python task before its interpreter and graphics are closed.
func (engine *Engine) Stop() {
	engine.Coroutine.Stop()
	engine.Coroutine.Wait()
}

func (engine *Engine) AddDrawCall(draw func(*ebiten.Image)) {
	engine.DrawCalls = append(engine.DrawCalls, draw)
}

func (engine *Engine) Update() error {
	/*
	   keys := inpututil.AppendJustPressedKeys(nil)
	   for _, key := range keys {
	       if key == ebiten.KeyEscape || key == ebiten.KeyCapsLock {
	           return ebiten.Termination
	       }
	   }
	*/

	// log.Printf("update")
	if engine.finished {
		if engine.finalFrameDrawn {
			return ebiten.Termination
		}
		return nil
	}
	previous := engine.DrawCalls
	engine.DrawCalls = nil
	err := engine.Coroutine.Run()
	if errors.Is(err, ebiten.Termination) {
		if len(engine.DrawCalls) == 0 {
			engine.DrawCalls = previous
		}
		if len(engine.DrawCalls) > 0 {
			// Ebitengine can Update several times before Draw. A short script
			// must still present its last flip before its Go runtime exits.
			engine.finished = true
			return nil
		}
	}
	return err
}

func (engine *Engine) SetScreenSize(width int, height int) {
	engine.ScreenWidth = width
	engine.ScreenHeight = height
}

func (engine *Engine) Draw(screen *ebiten.Image) {
	for _, draw := range engine.DrawCalls {
		draw(screen)
	}
	// set_mode may have changed Layout during the last Update. Wait for a
	// frame at the new logical size before stopping, avoiding a blank resize.
	if engine.finished && screen.Bounds().Dx() == engine.ScreenWidth && screen.Bounds().Dy() == engine.ScreenHeight {
		engine.finalFrameDrawn = true
	}
}

func (engine *Engine) Layout(outsideWidth int, outsideHeight int) (int, int) {
	return engine.ScreenWidth, engine.ScreenHeight
}

func testGraphics() error {
	var path string
	if len(os.Args) > 1 {
		path = os.Args[1]
	} else {
		return fmt.Errorf("usage: %s <script>", os.Args[0])
	}

	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("Python test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetTPS(20)

	opts := py.DefaultContextOpts()
	opts.SysArgs = os.Args[1:]

	python := py.NewContext(opts)
	defer python.Close()

	var engine *Engine
	var code *py.Code

	mainCoroutine := func(yield coroutine.YieldFunc) error {
		engine.Yield = yield

		_, err := py.RunCode(python, code, "python/"+path, py.MainModuleName)
		if err != nil {
			if py.IsException(py.SystemExit, err) {
				return ebiten.Termination
			}
			return err
		}

		return ebiten.Termination
	}

	engine = &Engine{
		Coroutine:    coroutine.MakeCoroutine(mainCoroutine),
		ScreenWidth:  640,
		ScreenHeight: 480,
	}

	defer engine.Stop()

	err := pygame.RegisterPygameModule(engine, python)
	if err != nil {
		return err
	}

	script, err := loadPythonSource(path)
	if err != nil {
		return fmt.Errorf("load Python script: %w", err)
	}

	code, err = py.Compile(string(script), path, "exec", 0, false)
	if err != nil {
		return err
	}

	return ebiten.RunGame(engine)
}

func main() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)
	// test1()
	if err := testGraphics(); err != nil {
		py.TracebackDump(err)
		os.Exit(1)
	}
}
