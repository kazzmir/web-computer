package scripts

import (
    "embed"
    "path/filepath"
)

//go:embed python/*
var Python embed.FS

func Load(path string) ([]byte, error) {
    return Python.ReadFile(filepath.Join("python", path))
}
