package py

func init() {
	TypeType.Dict["__bases__"] = &Property{Fget: func(self Object) (Object, error) { return self.(*Type).Bases.Copy(), nil }}
	TypeType.Dict["__mro__"] = &Property{Fget: func(self Object) (Object, error) { return self.(*Type).Mro.Copy(), nil }}
	TypeType.Dict["__base__"] = &Property{Fget: func(self Object) (Object, error) {
		base := self.(*Type).Base
		if base == nil {
			return None, nil
		}
		return base, nil
	}}
	TypeType.Dict["__name__"] = &Property{Fget: func(self Object) (Object, error) { return String(self.(*Type).Name), nil }}
	TypeType.Dict["__qualname__"] = &Property{Fget: func(self Object) (Object, error) {
		class := self.(*Type)
		if class.Qualname != "" {
			return String(class.Qualname), nil
		}
		return String(class.Name), nil
	}}
	ObjectType.Dict["__class__"] = &Property{Fget: func(self Object) (Object, error) { return self.Type(), nil }}
	dict := &Property{Fget: func(self Object) (Object, error) {
		if object, ok := self.(IGetDict); ok {
			return object.GetDict(), nil
		}
		return nil, ExceptionNewf(AttributeError, "'%s' object has no __dict__", self.Type().Name)
	}}
	ObjectType.Dict["__dict__"] = dict
	TypeType.Dict["__dict__"] = dict
	TypeType.Dict["mro"] = MustNewMethod("mro", func(self Object, args Tuple) (Object, error) {
		class, bound := self.(*Type)
		if !bound {
			var value Object
			if err := UnpackTuple(args, nil, "mro", 1, 1, &value); err != nil {
				return nil, err
			}
			var ok bool
			class, ok = value.(*Type)
			if !ok || (len(class.Mro) == 0 && class.Flags&TPFLAGS_READYING == 0) {
				return nil, ExceptionNewf(TypeError, "mro requires a type")
			}
		} else if err := UnpackTuple(args, nil, "mro", 0, 0); err != nil {
			return nil, err
		}
		return class.mro_implementation()
	}, 0, "Return the class method resolution order.")
}
