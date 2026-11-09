// Package operator provides rich-comparison and truth-testing functions.
package operator

import "github.com/kazzmir/web-computer/gpython/py"

func init() {
	methods := []*py.Method{}
	for _, entry := range []struct {
		name string
		fn   func(py.Object, py.Object) (py.Object, error)
	}{
		{"lt", py.Lt}, {"le", py.Le}, {"eq", py.Eq}, {"ne", py.Ne}, {"gt", py.Gt}, {"ge", py.Ge},
	} {
		name, operation := entry.name, entry.fn
		method := py.MustNewMethod(name, func(self py.Object, args py.Tuple) (py.Object, error) {
			var a, b py.Object
			if err := py.UnpackTuple(args, nil, name, 2, 2, &a, &b); err != nil {
				return nil, err
			}
			return operation(a, b)
		}, 0, "Apply the corresponding rich-comparison operator.")
		methods = append(methods, method)
	}
	methods = append(methods, py.MustNewMethod("not_", func(self, value py.Object) (py.Object, error) {
		truth, err := py.ObjectIsTrue(value)
		return py.Bool(!truth), err
	}, 0, "Return the logical negation of a value."))
	py.RegisterModule(&py.ModuleImpl{Info: py.ModuleInfo{Name: "operator", Doc: "Rich-comparison and logical-negation subset"}, Methods: methods, CodeSrc: `
__lt__ = lt
__le__ = le
__eq__ = eq
__ne__ = ne
__gt__ = gt
__ge__ = ge
__not__ = not_
`})
}
