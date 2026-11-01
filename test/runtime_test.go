package test

import (
	"testing"

	"github.com/kazzmir/web-computer/gpython/py"
	_ "github.com/kazzmir/web-computer/gpython/stdlib"
)

// Keep expected counts explicit so broken discovery cannot produce a false pass.
func TestCPython(t *testing.T) {
	for _, suite := range []struct {
		file  string
		count py.Int
	}{
		{"test_unary.py", 6},
		{"test_longexp.py", 1},
		{"test_keyword.py", 11},
		{"test_generator_stop.py", 2},
		{"test_int_literal.py", 6},
	} {
		t.Run(suite.file, func(t *testing.T) {
			ctx := py.NewContext(py.DefaultContextOpts())
			defer ctx.Close()
			if _, err := py.RunFile(ctx, suite.file, py.CompileOpts{}, nil); err != nil {
				py.TracebackDump(err)
				t.Fatal(err)
			}
			module, err := ctx.Store().GetModule("unittest")
			if err != nil {
				t.Fatal(err)
			}
			if got := module.Globals["_tests_run"]; got != suite.count {
				t.Fatalf("ran %v tests, want %d", got, suite.count)
			}
		})
	}
}
