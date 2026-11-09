package py

// Super is a cooperative MRO proxy. Zero-argument inference is performed by
// the VM, which has access to the calling frame's first argument/class cell.
type Super struct {
	This     *Type
	Self     Object
	SelfType *Type
}

var SuperType = NewTypeX("super", "Cooperative method-resolution-order proxy", superNew, nil)

func (s *Super) Type() *Type { return SuperType }
func superNew(t *Type, args Tuple, kwargs StringDict) (Object, error) {
	if len(args) == 0 {
		return nil, ExceptionNewf(RuntimeError, "super(): no arguments")
	}
	var clsObj Object
	var self Object = None
	if err := UnpackTuple(args, kwargs, "super", 1, 2, &clsObj, &self); err != nil {
		return nil, err
	}
	cls, ok := clsObj.(*Type)
	if !ok || !cls.Type().IsSubtype(TypeType) {
		return nil, ExceptionNewf(TypeError, "super() argument 1 must be a type")
	}
	proxy := &Super{This: cls, Self: self}
	if self != None {
		if typ, ok := self.(*Type); ok && typ.Type().IsSubtype(TypeType) && typ.IsSubtype(cls) {
			proxy.SelfType = typ
		} else if self.Type().IsSubtype(cls) {
			proxy.SelfType = self.Type()
		} else {
			return nil, ExceptionNewf(TypeError, "super(type, obj): obj must be an instance or subtype of type")
		}
	}
	return proxy, nil
}
func (s *Super) M__getattribute__(name string) (Object, error) {
	switch name {
	case "__class__":
		return SuperType, nil
	case "__thisclass__":
		return s.This, nil
	case "__self__":
		return s.Self, nil
	case "__self_class__":
		if s.SelfType != nil {
			return s.SelfType, nil
		}
		return None, nil
	}
	if s.SelfType != nil {
		found := false
		for _, entry := range s.SelfType.Mro {
			cls := entry.(*Type)
			if !found {
				if cls == s.This {
					found = true
				}
				continue
			}
			if value, ok := cls.Dict[name]; ok {
				if descriptor, ok := value.(I__get__); ok {
					instance := s.Self
					if s.Self == s.SelfType {
						instance = None
					}
					return descriptor.M__get__(instance, s.SelfType)
				}
				return value, nil
			}
		}
	}
	return nil, ExceptionNewf(AttributeError, "'super' object has no attribute '%s'", name)
}
