package py

type comparisonOp int

const (
	compareLT comparisonOp = iota
	compareLE
	compareEQ
	compareNE
	compareGT
	compareGE
)

var comparisonSymbols = [...]string{"<", "<=", "==", "!=", ">", ">="}
var reflectedComparison = [...]comparisonOp{compareGT, compareGE, compareEQ, compareNE, compareLT, compareLE}

func comparisonSlot(a, b Object, op comparisonOp) (Object, error) {
	switch op {
	case compareLT:
		if slot, ok := a.(I__lt__); ok {
			return slot.M__lt__(b)
		}
	case compareLE:
		if slot, ok := a.(I__le__); ok {
			return slot.M__le__(b)
		}
	case compareEQ:
		if slot, ok := a.(I__eq__); ok {
			return slot.M__eq__(b)
		}
	case compareNE:
		if slot, ok := a.(I__ne__); ok {
			return slot.M__ne__(b)
		}
	case compareGT:
		if slot, ok := a.(I__gt__); ok {
			return slot.M__gt__(b)
		}
	case compareGE:
		if slot, ok := a.(I__ge__); ok {
			return slot.M__ge__(b)
		}
	}
	return NotImplemented, nil
}

// Rich comparisons preserve arbitrary results. Only the caller deciding a
// truth value (e.g. if, bool(), container comparison) should coerce to bool.
func richCompare(a, b Object, op comparisonOp) (Object, error) {
	reverse := reflectedComparison[op]
	rightFirst := a.Type() != b.Type() && b.Type().IsSubtype(a.Type())
	if rightFirst {
		value, err := comparisonSlot(b, a, reverse)
		if err != nil || value != NotImplemented {
			return value, err
		}
	}
	value, err := comparisonSlot(a, b, op)
	if err != nil || value != NotImplemented {
		return value, err
	}
	if !rightFirst {
		value, err = comparisonSlot(b, a, reverse)
		if err != nil || value != NotImplemented {
			return value, err
		}
	}
	switch op {
	case compareEQ:
		return Bool(Identical(a, b)), nil
	case compareNE:
		return Bool(!Identical(a, b)), nil
	}
	return nil, ExceptionNewf(TypeError, "'%s' not supported between instances of '%s' and '%s'", comparisonSymbols[op], a.Type().Name, b.Type().Name)
}
func (t *Type) M__lt__(v Object) (Object, error) { return t.callBinarySlot("__lt__", v) }
func (t *Type) M__le__(v Object) (Object, error) { return t.callBinarySlot("__le__", v) }
func (t *Type) M__gt__(v Object) (Object, error) { return t.callBinarySlot("__gt__", v) }
func (t *Type) M__ge__(v Object) (Object, error) { return t.callBinarySlot("__ge__", v) }
