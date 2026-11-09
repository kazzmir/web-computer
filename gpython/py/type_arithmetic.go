package py

// callBinarySlot resolves special methods on the class, not the instance dict.
// Missing slots return NotImplemented; an explicitly non-callable slot must
// still raise TypeError rather than silently falling back to another operator.
func (t *Type) callBinarySlot(name string, other Object) (Object, error) {
	value, found, err := TypeCall1(t, name, other)
	if !found {
		return NotImplemented, nil
	}
	return value, err
}
func (t *Type) M__add__(v Object) (Object, error)       { return t.callBinarySlot("__add__", v) }
func (t *Type) M__radd__(v Object) (Object, error)      { return t.callBinarySlot("__radd__", v) }
func (t *Type) M__iadd__(v Object) (Object, error)      { return t.callBinarySlot("__iadd__", v) }
func (t *Type) M__sub__(v Object) (Object, error)       { return t.callBinarySlot("__sub__", v) }
func (t *Type) M__rsub__(v Object) (Object, error)      { return t.callBinarySlot("__rsub__", v) }
func (t *Type) M__isub__(v Object) (Object, error)      { return t.callBinarySlot("__isub__", v) }
func (t *Type) M__mul__(v Object) (Object, error)       { return t.callBinarySlot("__mul__", v) }
func (t *Type) M__rmul__(v Object) (Object, error)      { return t.callBinarySlot("__rmul__", v) }
func (t *Type) M__imul__(v Object) (Object, error)      { return t.callBinarySlot("__imul__", v) }
func (t *Type) M__truediv__(v Object) (Object, error)   { return t.callBinarySlot("__truediv__", v) }
func (t *Type) M__rtruediv__(v Object) (Object, error)  { return t.callBinarySlot("__rtruediv__", v) }
func (t *Type) M__itruediv__(v Object) (Object, error)  { return t.callBinarySlot("__itruediv__", v) }
func (t *Type) M__floordiv__(v Object) (Object, error)  { return t.callBinarySlot("__floordiv__", v) }
func (t *Type) M__rfloordiv__(v Object) (Object, error) { return t.callBinarySlot("__rfloordiv__", v) }
func (t *Type) M__ifloordiv__(v Object) (Object, error) { return t.callBinarySlot("__ifloordiv__", v) }
func (t *Type) M__mod__(v Object) (Object, error)       { return t.callBinarySlot("__mod__", v) }
func (t *Type) M__rmod__(v Object) (Object, error)      { return t.callBinarySlot("__rmod__", v) }
func (t *Type) M__imod__(v Object) (Object, error)      { return t.callBinarySlot("__imod__", v) }
func (t *Type) M__lshift__(v Object) (Object, error)    { return t.callBinarySlot("__lshift__", v) }
func (t *Type) M__rlshift__(v Object) (Object, error)   { return t.callBinarySlot("__rlshift__", v) }
func (t *Type) M__ilshift__(v Object) (Object, error)   { return t.callBinarySlot("__ilshift__", v) }
func (t *Type) M__rshift__(v Object) (Object, error)    { return t.callBinarySlot("__rshift__", v) }
func (t *Type) M__rrshift__(v Object) (Object, error)   { return t.callBinarySlot("__rrshift__", v) }
func (t *Type) M__irshift__(v Object) (Object, error)   { return t.callBinarySlot("__irshift__", v) }
func (t *Type) M__and__(v Object) (Object, error)       { return t.callBinarySlot("__and__", v) }
func (t *Type) M__rand__(v Object) (Object, error)      { return t.callBinarySlot("__rand__", v) }
func (t *Type) M__iand__(v Object) (Object, error)      { return t.callBinarySlot("__iand__", v) }
func (t *Type) M__or__(v Object) (Object, error) {
	if result, found, err := TypeCall1(t, "__or__", v); found {
		return result, err
	}
	if t.Type().IsSubtype(TypeType) {
		return NewUnion(t, v)
	}
	return NotImplemented, nil
}
func (t *Type) M__ror__(v Object) (Object, error) {
	if result, found, err := TypeCall1(t, "__ror__", v); found {
		return result, err
	}
	if t.Type().IsSubtype(TypeType) {
		return NewUnion(v, t)
	}
	return NotImplemented, nil
}
func (t *Type) M__ior__(v Object) (Object, error)  { return t.callBinarySlot("__ior__", v) }
func (t *Type) M__xor__(v Object) (Object, error)  { return t.callBinarySlot("__xor__", v) }
func (t *Type) M__rxor__(v Object) (Object, error) { return t.callBinarySlot("__rxor__", v) }
func (t *Type) M__ixor__(v Object) (Object, error) { return t.callBinarySlot("__ixor__", v) }
