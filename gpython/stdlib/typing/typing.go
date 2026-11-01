// Package typing provides runtime metadata for basic List annotations.
package typing

import "github.com/kazzmir/web-computer/gpython/py"

func init() {
	py.RegisterModule(&py.ModuleImpl{Info: py.ModuleInfo{Name: "typing", Doc: "Runtime List annotation metadata (limited compatibility subset)"}, CodeSrc: `
class _Alias:
    def __init__(self, origin, args):
        self.__origin__ = origin
        self.__args__ = args

class _Generic:
    def __init__(self, origin): self.__origin__ = origin
    def __getitem__(self, arg):
        return _Alias(self.__origin__, arg if isinstance(arg, tuple) else (arg,))

List = _Generic(list)
`})
}
