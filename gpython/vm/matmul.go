package vm

import "github.com/kazzmir/web-computer/gpython/py"

func do_BINARY_MATRIX_MULTIPLY(vm *Vm, arg int32) error {
	b := vm.POP()
	return vm.setTopAndCheckErr(py.MatMul(vm.TOP(), b))
}
func do_INPLACE_MATRIX_MULTIPLY(vm *Vm, arg int32) error {
	b := vm.POP()
	return vm.setTopAndCheckErr(py.IMatMul(vm.TOP(), b))
}
