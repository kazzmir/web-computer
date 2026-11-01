// Package unittest implements a small, fail-fast unittest compatibility subset.
package unittest

import (
	_ "embed"
	"regexp"
	"sort"
	"strings"

	"github.com/kazzmir/web-computer/gpython/py"
)

//go:embed unittest.py
var source string

func init() {
	py.RegisterModule(&py.ModuleImpl{
		Info:    py.ModuleInfo{Name: "unittest", FileDesc: "unittest.py", Doc: "Limited fail-fast TestCase assertions and module discovery"},
		CodeSrc: source,
		Methods: []*py.Method{
			py.MustNewMethod("_discover", discover, 0, "Discover TestCase methods in a loaded module."),
			py.MustNewMethod("_regex_matches", regexMatches, 0, "Search using Go/RE2 regular-expression syntax."),
		},
	})
}

func regexMatches(self py.Object, args py.Tuple) (py.Object, error) {
	var pattern, text py.Object
	if err := py.UnpackTuple(args, nil, "_regex_matches", 2, 2, &pattern, &text); err != nil {
		return nil, err
	}
	p, ok := pattern.(py.String)
	if !ok {
		return nil, py.ExceptionNewf(py.TypeError, "pattern must be a string")
	}
	s, ok := text.(py.String)
	if !ok {
		return nil, py.ExceptionNewf(py.TypeError, "text must be a string")
	}
	re, err := regexp.Compile(string(p))
	if err != nil {
		return nil, py.ExceptionNewf(py.ValueError, "unsupported or invalid regex: %v", err)
	}
	return py.NewBool(re.MatchString(string(s))), nil
}

func discover(self py.Object, args py.Tuple) (py.Object, error) {
	var arg py.Object
	if err := py.UnpackTuple(args, nil, "_discover", 1, 1, &arg); err != nil {
		return nil, err
	}
	name, ok := arg.(py.String)
	if !ok {
		return nil, py.ExceptionNewf(py.TypeError, "module name must be a string")
	}
	unittest := self.(*py.Module)
	module, err := unittest.Context.Store().GetModule(string(name))
	if err != nil {
		return nil, err
	}
	base := unittest.Globals["TestCase"].(*py.Type)
	classes := []string{}
	for name, object := range module.Globals {
		if class, ok := object.(*py.Type); ok && class != base && class.IsSubtype(base) {
			classes = append(classes, name)
		}
	}
	sort.Strings(classes)
	tests := py.NewList()
	for _, name := range classes {
		class := module.Globals[name].(*py.Type)
		names := map[string]bool{}
		for _, parent := range class.Mro {
			for method := range parent.(*py.Type).Dict {
				if strings.HasPrefix(method, "test") {
					names[method] = true
				}
			}
		}
		methods := []string{}
		for method := range names {
			attr, err := py.GetAttrString(class, method)
			if err != nil {
				return nil, err
			}
			if _, ok := attr.(py.I__call__); ok {
				methods = append(methods, method)
			}
		}
		sort.Strings(methods)
		for _, method := range methods {
			tests.Items = append(tests.Items, py.Tuple{class, py.String(method)})
		}
	}
	return tests, nil
}
