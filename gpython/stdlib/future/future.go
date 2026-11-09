// Package future exposes metadata for supported future imports.
package future

import "github.com/kazzmir/web-computer/gpython/py"

func init() {
	py.RegisterModule(&py.ModuleImpl{
		Info: py.ModuleInfo{Name: "__future__", FileDesc: "__future__.py", Doc: "Future metadata for generator_stop and barry_as_FLUFL"},
		CodeSrc: `
# Other future features are deliberately not advertised as supported.
__all__ = ['generator_stop', 'barry_as_FLUFL']
all_feature_names = ['generator_stop', 'barry_as_FLUFL']
CO_FUTURE_GENERATOR_STOP = 0x800000
CO_FUTURE_BARRY_AS_BDFL = 0x400000
class _Feature:
    def __init__(self, optional, mandatory, compiler_flag):
        self.optional = optional
        self.mandatory = mandatory
        self.compiler_flag = compiler_flag
    def getOptionalRelease(self): return self.optional
    def getMandatoryRelease(self): return self.mandatory

generator_stop = _Feature((3, 5, 0, 'beta', 1), (3, 7, 0, 'alpha', 0), CO_FUTURE_GENERATOR_STOP)
barry_as_FLUFL = _Feature((3, 1, 0, 'alpha', 2), (4, 0, 0, 'alpha', 0), CO_FUTURE_BARRY_AS_BDFL)
`,
	})
}
