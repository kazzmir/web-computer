package py

func (a *List) compare(other Object, op comparisonOp) (Object, error) {
	b, ok := other.(*List)
	if !ok {
		return NotImplemented, nil
	}
	if a == b {
		switch op {
		case compareEQ, compareLE, compareGE:
			return True, nil
		default:
			return False, nil
		}
	}
	if (op == compareEQ || op == compareNE) && len(a.Items) != len(b.Items) {
		return Bool(op == compareNE), nil
	}
	for i := 0; i < len(a.Items) && i < len(b.Items); i++ {
		left, right := a.Items[i], b.Items[i]
		equal := Identical(left, right)
		if !equal {
			result, err := Eq(left, right)
			if err != nil {
				return nil, err
			}
			equal, err = ObjectIsTrue(result)
			if err != nil {
				return nil, err
			}
		}
		if equal {
			continue
		}
		switch op {
		case compareEQ:
			return False, nil
		case compareNE:
			return True, nil
		default:
			return richCompare(left, right, op)
		}
	}
	left, right := len(a.Items), len(b.Items)
	switch op {
	case compareEQ:
		return Bool(left == right), nil
	case compareNE:
		return Bool(left != right), nil
	case compareLT:
		return Bool(left < right), nil
	case compareLE:
		return Bool(left <= right), nil
	case compareGT:
		return Bool(left > right), nil
	default:
		return Bool(left >= right), nil
	}
}
func (a *List) M__lt__(v Object) (Object, error) { return a.compare(v, compareLT) }
func (a *List) M__le__(v Object) (Object, error) { return a.compare(v, compareLE) }
func (a *List) M__gt__(v Object) (Object, error) { return a.compare(v, compareGT) }
func (a *List) M__ge__(v Object) (Object, error) { return a.compare(v, compareGE) }
