package main

import (
    "log"

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

func registerPygameModule(yielder *PygameYield) {
    pygame_draw := func(self py.Object, args py.Tuple) (py.Object, error) {
        yielder.Yield()
        return py.None, nil
    }

    methods := []*py.Method{
        py.MustNewMethod("draw", pygame_draw, 0, ""),
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
}

func (engine *Engine) Update() error {
    keys := inpututil.AppendJustPressedKeys(nil)
    for _, key := range keys {
        if key == ebiten.KeyEscape || key == ebiten.KeyCapsLock {
            return ebiten.Termination
        }
    }

    engine.Coroutine.Run()

    return nil
}

func (engine *Engine) Draw(screen *ebiten.Image) {
    log.Printf("draw!")
}

func (engine *Engine) Layout(outsideWidth int, outsideHeight int) (int, int) {
    return 640, 480
}

type PygameYield struct {
    Yield coroutine.YieldFunc
}

func testGraphics() {
    ebiten.SetWindowSize(800, 600)
    ebiten.SetWindowTitle("Python test")
    ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

    opts := py.DefaultContextOpts()

    python := py.NewContext(opts)
    defer python.Close()

    var yielder PygameYield

    registerPygameModule(&yielder)

    script, err := scripts.Load("game1.py")
    if err != nil {
        log.Printf("Unable to load python script: %v", err)
        return
    }

    registerWebTestModule()

    code, err := py.Compile(string(script), "game1.py", "exec", 0, false)
    if err != nil {
        log.Printf("Unable to compile python script: %v", err)
        return
    }

    mainCoroutine := func (yield coroutine.YieldFunc) error {
        yielder.Yield = yield

        module, err := py.RunCode(python, code, "script", "user")
        if err != nil {
            log.Printf("Unable to run code: %v", err)
        } else {
            log.Printf("Module: %v", module)
        }

        return ebiten.Termination
    }

    engine := &Engine{
        Coroutine: coroutine.MakeCoroutine(mainCoroutine),
    }

    err = ebiten.RunGame(engine)
    if err != nil {
        log.Printf("Error: %v", err)
    }
}

func main() {
    test1()
    testGraphics()
}
