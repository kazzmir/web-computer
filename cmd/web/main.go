package main

import (
    "log"
    "image/color"

    "github.com/kazzmir/web-computer/scripts"
    "github.com/kazzmir/web-computer/lib/coroutine"

    "github.com/go-python/gpython/py"
    _ "github.com/go-python/gpython/stdlib"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/inpututil"
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
            Doc: "webtest module",
        },
        Methods: methods,
        Globals: py.StringDict{},
        OnContextClosed: func (instance *py.Module) {
        },
    })
}

func registerPygameModule(engine *Engine) {
    pygame_draw := func (self py.Object, args py.Tuple) (py.Object, error) {
        engine.Yield()
        return py.None, nil
    }

    pygame_clear := func (self py.Object, args py.Tuple) (py.Object, error) {
        var value py.Int
        err := py.LoadTuple(args, []interface{}{&value})
        if err != nil {
            log.Printf("Unable to load args: %v", err)
            return py.None, err
        }

        num, err := value.GoInt()
        if err != nil {
            log.Printf("Unable to convert to int: %v", err)
            return py.None, err
        }

        engine.AddDrawCall(func (screen *ebiten.Image) {
            var c color.Color
            switch num {
                case 0: c = color.RGBA{R: 255, G: 0, B: 0, A: 255}
                case 1: c = color.RGBA{R: 0, G: 255, B: 0, A: 255}
                case 2: c = color.RGBA{R: 0, G: 0, B: 255, A: 255}
                case 3: c = color.RGBA{R: 255, G: 255, B: 0, A: 255}
                default: c = color.RGBA{R: 0, G: 0, B: 0, A: 255}
            }

            screen.Fill(c)
        })

        return py.None, nil
    }

    methods := []*py.Method{
        py.MustNewMethod("draw", pygame_draw, 0, ""),
        py.MustNewMethod("clear", pygame_clear, 0, "clear screen to a color"),
    }

    py.RegisterModule(&py.ModuleImpl{
        Info: py.ModuleInfo{
            Name: "pygame",
            Doc: "pygame module",
        },
        Methods: methods,
        Globals: py.StringDict{},
        OnContextClosed: func (instance *py.Module) {
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
    Yield coroutine.YieldFunc
    DrawCalls []func(*ebiten.Image)
}

func (engine *Engine) AddDrawCall(draw func(*ebiten.Image)) {
    engine.DrawCalls = append(engine.DrawCalls, draw)
}

func (engine *Engine) Update() error {
    keys := inpututil.AppendJustPressedKeys(nil)
    for _, key := range keys {
        if key == ebiten.KeyEscape || key == ebiten.KeyCapsLock {
            return ebiten.Termination
        }
    }

    log.Printf("update")
    engine.DrawCalls = nil
    engine.Coroutine.Run()

    return nil
}

func (engine *Engine) Draw(screen *ebiten.Image) {
    for _, draw := range engine.DrawCalls {
        draw(screen)
    }
}

func (engine *Engine) Layout(outsideWidth int, outsideHeight int) (int, int) {
    return 640, 480
}

func testGraphics() {
    ebiten.SetWindowSize(800, 600)
    ebiten.SetWindowTitle("Python test")
    ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
    ebiten.SetTPS(20)

    opts := py.DefaultContextOpts()

    python := py.NewContext(opts)
    defer python.Close()

    var engine *Engine
    var code *py.Code

    mainCoroutine := func (yield coroutine.YieldFunc) error {
        engine.Yield = yield

        module, err := py.RunCode(python, code, "script", "user")
        if err != nil {
            log.Printf("Unable to run code: %v", err)
        } else {
            log.Printf("Module: %v", module)
        }

        return ebiten.Termination
    }

    engine = &Engine{
        Coroutine: coroutine.MakeCoroutine(mainCoroutine),
    }

    registerPygameModule(engine)

    script, err := scripts.Load("game1.py")
    if err != nil {
        log.Printf("Unable to load python script: %v", err)
        return
    }

    code, err = py.Compile(string(script), "game1.py", "exec", 0, false)
    if err != nil {
        log.Printf("Unable to compile python script: %v", err)
        return
    }

    err = ebiten.RunGame(engine)
    if err != nil {
        log.Printf("Error: %v", err)
    }
}

func main() {
    log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)
    test1()
    testGraphics()
}
