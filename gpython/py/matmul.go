package py

import "reflect"

func sameOperatorSlot(a, b Object) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	av, bv := reflect.ValueOf(a), reflect.ValueOf(b)
	if av.Type() != bv.Type() {
		return false
	}
	if av.Comparable() {
		return av.Equal(bv)
	}
	// Inherited non-callable slice/map objects still have identity. They
	// will raise TypeError if selected, but must not panic during dispatch.
	switch av.Kind() {
	case reflect.Slice:
		return av.Pointer() == bv.Pointer()
	}
	return false
}

// MatMul implements PEP 465's operator protocol, not a particular matrix type.
func MatMul(a, b Object) (Object, error) {
	left, right := a.Type(), b.Type()
	reflectedFirst := left != right && right.IsSubtype(left) && !sameOperatorSlot(right.Lookup("__rmatmul__"), left.Lookup("__rmatmul__"))
	call := func(self Object, name string, other Object) (Object, error) {
		value, found, err := TypeCall1(self, name, other)
		if !found {
			return NotImplemented, nil
		}
		return value, err
	}
	if reflectedFirst {
		value, err := call(b, "__rmatmul__", a)
		if err != nil || value != NotImplemented {
			return value, err
		}
	}
	value, err := call(a, "__matmul__", b)
	if err != nil || value != NotImplemented {
		return value, err
	}
	if left != right && !reflectedFirst {
		value, err = call(b, "__rmatmul__", a)
		if err != nil || value != NotImplemented {
			return value, err
		}
	}
	return nil, ExceptionNewf(TypeError, "unsupported operand type(s) for @: '%s' and '%s'", left.Name, right.Name)
}
func IMatMul(a, b Object) (Object, error) {
	value, found, err := TypeCall1(a, "__imatmul__", b)
	if err != nil {
		return nil, err
	}
	if found && value != NotImplemented {
		return value, nil
	}
	return MatMul(a, b)
}
