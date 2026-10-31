package os

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/kazzmir/web-computer/gpython/py"
)

// Path operations preserve separators and trailing slashes where Python does;
// filepath.Join and filepath.Split clean paths differently from os.path.
const pathSource = `
import os
sep = os.sep
altsep = os.altsep
curdir = '.'
pardir = '..'
extsep = '.'
pathsep = os.pathsep

def join(a, *parts):
    for b in parts:
        if isabs(b):
            a = b
        elif not a or a.endswith(sep):
            a = a + b
        else:
            a = a + sep + b
    return a

def split(p):
    i = len(p)
    while i and p[i-1] != sep:
        i -= 1
    head, tail = p[:i], p[i:]
    if head and head != sep * len(head):
        head = head.rstrip(sep)
    return head, tail

def dirname(p):
    return split(p)[0]

def basename(p):
    return split(p)[1]

def abspath(p):
    if not isabs(p):
        p = join(os.getcwd(), p)
    return normpath(p)
`

func init() {
	unary := func(fn func(string) (py.Object, error)) func(py.Object, py.Tuple) (py.Object, error) {
		return func(self py.Object, args py.Tuple) (py.Object, error) {
			var p py.Object
			if err := py.ParseTupleAndKeywords(args, nil, "s", []string{"path"}, &p); err != nil {
				return nil, err
			}
			return fn(string(p.(py.String)))
		}
	}
	py.RegisterModule(&py.ModuleImpl{
		Info:    py.ModuleInfo{Name: "os.path", Doc: "Filesystem path operations"},
		CodeSrc: pathSource,
		Methods: []*py.Method{
			py.MustNewMethod("isabs", unary(func(p string) (py.Object, error) { return py.Bool(filepath.IsAbs(p)), nil }), 0, ""),
			py.MustNewMethod("normpath", unary(func(p string) (py.Object, error) {
				result := filepath.Clean(p)
				if os.PathSeparator == '/' && strings.HasPrefix(p, "//") && !strings.HasPrefix(p, "///") {
					result = "/" + result
				}
				return py.String(result), nil
			}), 0, ""),
			py.MustNewMethod("exists", unary(func(p string) (py.Object, error) { _, err := os.Stat(p); return py.Bool(err == nil), nil }), 0, ""),
			py.MustNewMethod("isfile", unary(func(p string) (py.Object, error) {
				s, e := os.Stat(p)
				return py.Bool(e == nil && s.Mode().IsRegular()), nil
			}), 0, ""),
			py.MustNewMethod("isdir", unary(func(p string) (py.Object, error) { s, e := os.Stat(p); return py.Bool(e == nil && s.IsDir()), nil }), 0, ""),
		},
	})
}
