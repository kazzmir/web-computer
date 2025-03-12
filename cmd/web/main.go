package main

import (
    "log"

    "github.com/kazzmir/web-computer/scripts"

    "github.com/go-python/gpython/py"
    _ "github.com/go-python/gpython/stdlib"
)

func webtest_foo(self py.Object, args py.Tuple) (py.Object, error) {
    log.Printf("Invoking go foo from python")
    return py.String("foo"), nil
}

func registerModule() {
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

func main() {
    log.Printf("Starting python test")
    opts := py.DefaultContextOpts()

    python := py.NewContext(opts)
    defer python.Close()

    script, err := scripts.Load("xmod.py")
    if err != nil {
        log.Printf("Unable to load python script: %v", err)
        return
    }

    registerModule()

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
