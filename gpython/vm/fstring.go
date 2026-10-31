package vm

import (
	"strings"

	"github.com/kazzmir/web-computer/gpython/py"
)

func do_FORMAT_VALUE(vm *Vm, flags int32) error {
	spec := py.String("")
	if flags&4 != 0 {
		value := vm.POP()
		var ok bool
		spec, ok = value.(py.String)
		if !ok {
			return py.ExceptionNewf(py.TypeError, "format specifier must be a string")
		}
	}
	value := vm.TOP()
	var err error
	switch flags & 3 {
	case 1:
		value, err = py.Str(value)
	case 2:
		value, err = py.Repr(value)
	case 3:
		value, err = py.Ascii(value)
	}
	if err != nil {
		return err
	}
	if flags&3 != 0 {
		if _, ok := value.(py.String); !ok {
			return py.ExceptionNewf(py.TypeError, "f-string conversion must return a string")
		}
	}
	return vm.setTopAndCheckErr(py.Format(value, spec))
}

func do_BUILD_STRING(vm *Vm, count int32) error {
	var out strings.Builder
	for _, part := range vm.frame.Stack[len(vm.frame.Stack)-int(count):] {
		text, ok := part.(py.String)
		if !ok {
			return py.ExceptionNewf(py.TypeError, "f-string part must be a string")
		}
		out.WriteString(string(text))
	}
	vm.DROPN(int(count))
	vm.PUSH(py.String(out.String()))
	return nil
}
