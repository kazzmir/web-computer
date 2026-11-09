package py

import "strings"

type GenericAlias struct {
	Origin *Type
	Args   Tuple
	Name   string
}

var GenericAliasType = NewType("GenericAlias", "Parameterized runtime type metadata")

func (g *GenericAlias) Type() *Type { return GenericAliasType }
func NewGenericAlias(origin *Type, key Object, name string) *GenericAlias {
	args, ok := key.(Tuple)
	if !ok {
		args = Tuple{key}
	}
	return &GenericAlias{Origin: origin, Args: args.Copy(), Name: name}
}
func (g *GenericAlias) M__or__(value Object) (Object, error)  { return NewUnion(g, value) }
func (g *GenericAlias) M__ror__(value Object) (Object, error) { return NewUnion(value, g) }
func (g *GenericAlias) M__repr__() (Object, error) {
	name := g.Name
	if name == "" {
		name = g.Origin.Name
	}
	parts := []string{}
	for _, arg := range g.Args {
		part, err := unionName(arg)
		if err != nil {
			return nil, err
		}
		parts = append(parts, part)
	}
	return String(name + "[" + strings.Join(parts, ", ") + "]"), nil
}
func (g *GenericAlias) M__eq__(other Object) (Object, error) {
	v, ok := other.(*GenericAlias)
	if !ok {
		return NotImplemented, nil
	}
	if g.Origin != v.Origin {
		return False, nil
	}
	return g.Args.M__eq__(v.Args)
}
func (g *GenericAlias) M__ne__(other Object) (Object, error) { return notEq(g.M__eq__(other)) }

// TypingAlias is an unsubscripted compatibility alias such as typing.List.
// Parameterization produces GenericAlias metadata, not a runtime-checkable type.
type TypingAlias struct {
	Origin *Type
	Name   string
}

var TypingAliasType = NewType("_SpecialGenericAlias", "Unsubscripted typing alias")

func (g *TypingAlias) Type() *Type { return TypingAliasType }
func (g *TypingAlias) M__getitem__(key Object) (Object, error) {
	return NewGenericAlias(g.Origin, key, g.Name), nil
}
func (g *TypingAlias) M__or__(value Object) (Object, error)  { return NewUnion(g, value) }
func (g *TypingAlias) M__ror__(value Object) (Object, error) { return NewUnion(value, g) }
func (g *TypingAlias) M__repr__() (Object, error)            { return String(g.Name), nil }
func init() {
	GenericAliasType.Dict["__origin__"] = &Property{Fget: func(self Object) (Object, error) { return self.(*GenericAlias).Origin, nil }}
	GenericAliasType.Dict["__args__"] = &Property{Fget: func(self Object) (Object, error) { return self.(*GenericAlias).Args, nil }}
	TypingAliasType.Dict["__origin__"] = &Property{Fget: func(self Object) (Object, error) { return self.(*TypingAlias).Origin, nil }}
	for _, origin := range []*Type{ListType, TupleType, StringDictType, SetType, FrozenSetType} {
		origin.Dict["__class_getitem__"] = &ClassMethod{Callable: MustNewMethod("__class_getitem__", func(self Object, args Tuple) (Object, error) {
			var key Object
			if err := UnpackTuple(args, nil, "__class_getitem__", 1, 1, &key); err != nil {
				return nil, err
			}
			return NewGenericAlias(self.(*Type), key, ""), nil
		}, 0, "Create parameterized type metadata."), Dict: NewStringDict()}
	}
}
