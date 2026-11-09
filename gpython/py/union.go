package py

import "strings"

type Union struct{ Args Tuple }

var UnionType = NewType("Union", "Runtime union of types")

func (u *Union) Type() *Type                           { return UnionType }
func (u *Union) M__or__(other Object) (Object, error)  { return NewUnion(u, other) }
func (u *Union) M__ror__(other Object) (Object, error) { return NewUnion(other, u) }
func init() {
	UnionType.Dict["__args__"] = &Property{Fget: func(self Object) (Object, error) { return self.(*Union).Args, nil }}
	UnionType.Dict["__origin__"] = &Property{Fget: func(self Object) (Object, error) { return UnionType, nil }}
}
func unionOperand(value Object) bool {
	switch value.(type) {
	case *Union, *GenericAlias, *TypingAlias:
		return true
	}
	if value == None {
		return true
	}
	if class, ok := value.(*Type); ok {
		return class.Type().IsSubtype(TypeType)
	}
	return false
}
func NewUnion(left, right Object) (Object, error) {
	if !unionOperand(left) || !unionOperand(right) {
		return NotImplemented, nil
	}
	args := Tuple{}
	var add func(Object) error
	add = func(value Object) error {
		if value == None {
			value = NoneTypeType
		}
		if nested, ok := value.(*Union); ok {
			for _, arg := range nested.Args {
				if err := add(arg); err != nil {
					return err
				}
			}
			return nil
		}
		for _, arg := range args {
			equal := Identical(arg, value)
			if a, ok := arg.(*GenericAlias); ok {
				if b, ok := value.(*GenericAlias); ok {
					result, err := a.M__eq__(b)
					if err != nil {
						return err
					}
					equal, err = ObjectIsTrue(result)
					if err != nil {
						return err
					}
				}
			}
			if equal {
				return nil
			}
		}
		args = append(args, value)
		return nil
	}
	if err := add(left); err != nil {
		return nil, err
	}
	if err := add(right); err != nil {
		return nil, err
	}
	if len(args) == 1 {
		return args[0], nil
	}
	return &Union{Args: args}, nil
}
func unionName(value Object) (string, error) {
	if value == NoneTypeType {
		return "None", nil
	}
	if class, ok := value.(*Type); ok && class.Type().IsSubtype(TypeType) {
		return class.QualnameOrName(), nil
	}
	return ReprAsString(value)
}
func (t *Type) QualnameOrName() string {
	if t.Qualname != "" {
		return t.Qualname
	}
	return t.Name
}
func (u *Union) M__repr__() (Object, error) {
	parts := []string{}
	for _, arg := range u.Args {
		part, err := unionName(arg)
		if err != nil {
			return nil, err
		}
		parts = append(parts, part)
	}
	return String(strings.Join(parts, " | ")), nil
}
func (u *Union) M__eq__(other Object) (Object, error) {
	v, ok := other.(*Union)
	if !ok {
		return NotImplemented, nil
	}
	if len(u.Args) != len(v.Args) {
		return False, nil
	}
	for _, a := range u.Args {
		found := false
		for _, b := range v.Args {
			equal, err := Eq(a, b)
			if err != nil {
				return nil, err
			}
			truth, err := ObjectIsTrue(equal)
			if err != nil {
				return nil, err
			}
			if truth {
				found = true
				break
			}
		}
		if !found {
			return False, nil
		}
	}
	return True, nil
}
func (u *Union) M__ne__(other Object) (Object, error) { return notEq(u.M__eq__(other)) }
