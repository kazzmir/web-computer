package main

import (
    "log"

    "github.com/kazzmir/web-computer/scripts"

    "github.com/go-python/gpython/py"
    _ "github.com/go-python/gpython/stdlib"
)

func main() {
    log.Printf("Starting python test")
    opts := py.DefaultContextOpts()

    python := py.NewContext(opts)
    defer python.Close()

    script, err := scripts.Load("test1.py")
    if err != nil {
        log.Printf("Unable to load python script: %v", err)
        return
    }

    code, err := py.Compile(string(script), "test1.py", "exec", 0, false)
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
