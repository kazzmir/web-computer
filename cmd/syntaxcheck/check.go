package main

import (
	"fmt"
	"strings"

	"github.com/kazzmir/web-computer/gpython/compile"
	"github.com/kazzmir/web-computer/gpython/py"
)

type diagnostic struct {
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Message string `json:"message"`
}

// Compile, never run: this also detects invalid return/break statements and
// other errors that a token-only or grammar-only check would miss.
func checkSource(source string) (result []diagnostic, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("syntax checker: %v", recovered)
		}
	}()
	_, err = compile.Compile(source, "editor.py", py.ExecMode, 0, false)
	if err == nil {
		return []diagnostic{}, nil
	}
	if !py.IsException(py.SyntaxError, err) {
		return nil, err
	}
	exception := py.MakeException(err)
	line, column := 1, 1
	if value, ok := exception.Dict["lineno"].(py.Int); ok {
		line = int(value)
	}
	if value, ok := exception.Dict["offset"].(py.Int); ok {
		column = int(value)
	}
	// EOF errors may point just past the final source line. Keep the marker on
	// an existing gutter line while retaining the compiler's error description.
	lines := strings.Count(source, "\n") + 1
	line = max(1, min(lines, line))
	column = max(1, column)
	message := exception.Type().Name
	if args, ok := exception.Args.(py.Tuple); ok && len(args) > 0 {
		message += ": " + fmt.Sprint(args[0])
	}
	return []diagnostic{{Line: line, Column: column, Message: message}}, nil
}
