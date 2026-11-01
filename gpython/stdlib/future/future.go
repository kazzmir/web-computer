// Package future exposes metadata for supported future imports.
package future

import "github.com/kazzmir/web-computer/gpython/py"

func init() {
	py.RegisterModule(&py.ModuleImpl{
		Info: py.ModuleInfo{Name: "__future__", FileDesc: "__future__.py", Doc: "Limited future metadata: generator_stop is mandatory in this runtime"},
		CodeSrc: `
# Other future features are deliberately not advertised as supported.
__all__ = ['generator_stop']
all_feature_names = ['generator_stop']
CO_FUTURE_GENERATOR_STOP = 0x800000
class _Feature:
    def __init__(self, optional, mandatory, compiler_flag):
        self.optional = optional
        self.mandatory = mandatory
        self.compiler_flag = compiler_flag
    def getOptionalRelease(self): return self.optional
    def getMandatoryRelease(self): return self.mandatory

generator_stop = _Feature((3, 5, 0, 'beta', 1), (3, 7, 0, 'alpha', 0), CO_FUTURE_GENERATOR_STOP)
`,
	})
}
