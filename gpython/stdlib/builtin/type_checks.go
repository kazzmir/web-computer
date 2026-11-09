package builtin

import "github.com/kazzmir/web-computer/gpython/py"

func realClass(value py.Object) (*py.Type, bool) {
	class, ok := value.(*py.Type)
	return class, ok && class.Type().IsSubtype(py.TypeType)
}
func classBases(value py.Object) (py.Tuple, bool, error) {
	bases, err := py.GetAttrString(value, "__bases__")
	if err != nil {
		if py.IsException(py.AttributeError, err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	tuple, ok := bases.(py.Tuple)
	return tuple, ok, nil
}
func typeCheckBudget(self py.Object) int {
	if module, ok := self.(*py.Module); ok && module != nil {
		remaining := module.Context.Store().RecursionLimit() - module.Context.Store().RecursionDepth()
		if remaining < 1 {
			return 1
		}
		return remaining
	}
	return 1000
}
func checkDepth(depth, budget int) error {
	if depth >= budget {
		return py.ExceptionNewf(py.RecursionError, "maximum recursion depth exceeded in type check")
	}
	return nil
}
func abstractSubclass(source, target py.Object, depth, budget int) (py.Bool, error) {
	if err := checkDepth(depth, budget); err != nil {
		return false, err
	}
	if py.Identical(source, target) {
		return true, nil
	}
	if a, aok := realClass(source); aok {
		if b, bok := realClass(target); bok {
			return py.Bool(a.IsSubtype(b)), nil
		}
	}
	bases, ok, err := classBases(source)
	if err != nil || !ok {
		return false, err
	}
	for _, base := range bases {
		result, err := abstractSubclass(base, target, depth+1, budget)
		if err != nil || result {
			return result, err
		}
	}
	return false, nil
}
func instanceCheck(obj, info py.Object, depth, budget int) (py.Bool, error) {
	if err := checkDepth(depth, budget); err != nil {
		return false, err
	}
	switch target := info.(type) {
	case py.Tuple:
		for _, item := range target {
			result, err := instanceCheck(obj, item, depth+1, budget)
			if err != nil || result {
				return result, err
			}
		}
		return false, nil
	case *py.Union:
		return instanceCheck(obj, target.Args, depth+1, budget)
	case *py.GenericAlias:
		return false, py.ExceptionNewf(py.TypeError, "isinstance() argument 2 cannot be a parameterized generic")
	case *py.TypingAlias:
		return instanceCheck(obj, target.Origin, depth+1, budget)
	}
	target, isClass := realClass(info)
	if isClass && obj.Type() == target {
		return true, nil
	}
	if result, found, err := py.TypeCall1(info, "__instancecheck__", obj); found {
		if err != nil {
			return false, err
		}
		truth, err := py.ObjectIsTrue(result)
		return py.Bool(truth), err
	}
	if !isClass {
		_, valid, err := classBases(info)
		if err != nil {
			return false, err
		}
		if !valid {
			return false, py.ExceptionNewf(py.TypeError, "isinstance() arg 2 must be a type, a tuple of types, or a union")
		}
	}
	if isClass && obj.Type().IsSubtype(target) {
		return true, nil
	}
	apparent, err := py.GetAttrString(obj, "__class__")
	if err != nil {
		if py.IsException(py.AttributeError, err) {
			return false, nil
		}
		return false, err
	}
	return abstractSubclass(apparent, info, depth+1, budget)
}
func subclassCheck(candidate, info py.Object, depth, budget int) (py.Bool, error) {
	if err := checkDepth(depth, budget); err != nil {
		return false, err
	}
	switch target := info.(type) {
	case py.Tuple:
		for _, item := range target {
			result, err := subclassCheck(candidate, item, depth+1, budget)
			if err != nil || result {
				return result, err
			}
		}
		return false, nil
	case *py.Union:
		return subclassCheck(candidate, target.Args, depth+1, budget)
	case *py.GenericAlias:
		return false, py.ExceptionNewf(py.TypeError, "issubclass() argument 2 cannot be a parameterized generic")
	case *py.TypingAlias:
		if source, ok := candidate.(*py.TypingAlias); ok {
			candidate = source.Origin
		}
		return subclassCheck(candidate, target.Origin, depth+1, budget)
	}
	if result, found, err := py.TypeCall1(info, "__subclasscheck__", candidate); found {
		if err != nil {
			return false, err
		}
		truth, err := py.ObjectIsTrue(result)
		return py.Bool(truth), err
	}
	if _, valid := realClass(candidate); !valid {
		_, ok, err := classBases(candidate)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, py.ExceptionNewf(py.TypeError, "issubclass() arg 1 must be a class")
		}
	}
	if _, valid := realClass(info); !valid {
		_, ok, err := classBases(info)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, py.ExceptionNewf(py.TypeError, "issubclass() arg 2 must be a class, a tuple of classes, or a union")
		}
	}
	return abstractSubclass(candidate, info, depth+1, budget)
}
