package py

// Python-defined instances use Type wrappers rather than native Go numeric
// interfaces. Bridge their power slots for both ** and builtin pow().
func (t *Type) M__pow__(other, modulus Object) (Object, error) {
	args := Tuple{t, other}
	if modulus != None {
		args = append(args, modulus)
	}
	value, found, err := TypeCall(t, "__pow__", args, nil)
	if !found {
		return NotImplemented, nil
	}
	return value, err
}
func (t *Type) M__rpow__(other Object) (Object, error) {
	value, found, err := TypeCall1(t, "__rpow__", other)
	if !found {
		return NotImplemented, nil
	}
	return value, err
}
func (t *Type) M__ipow__(other, modulus Object) (Object, error) {
	args := Tuple{t, other}
	if modulus != None {
		args = append(args, modulus)
	}
	value, found, err := TypeCall(t, "__ipow__", args, nil)
	if !found {
		return NotImplemented, nil
	}
	return value, err
}
