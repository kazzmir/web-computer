package py

import "reflect"

func pythonConstructorSlot(value Object) Object {
	if _, ok := value.(*Function); ok {
		return value
	}
	if static, ok := value.(*StaticMethod); ok {
		if _, ok := static.Callable.(*Function); ok {
			return static.Callable
		}
	}
	return nil
}

func init() {
	ObjectType.Dict["__new__"] = &StaticMethod{Callable: MustNewMethod("__new__", func(self Object, args Tuple, kwargs StringDict) (Object, error) {
		if len(args) == 0 {
			return nil, ExceptionNewf(TypeError, "object.__new__ needs a type")
		}
		cls, ok := args[0].(*Type)
		if !ok || !cls.Type().IsSubtype(TypeType) {
			return nil, ExceptionNewf(TypeError, "object.__new__ argument 1 must be a type")
		}
		if cls.New == nil || reflect.ValueOf(cls.New).Pointer() != reflect.ValueOf(ObjectNew).Pointer() {
			return nil, ExceptionNewf(TypeError, "object.__new__(%s) is not safe; use %s.__new__()", cls.Name, cls.Name)
		}
		return ObjectNew(cls, args[1:], kwargs)
	}, 0, "Allocate an object."), Dict: NewStringDict()}
	ObjectType.Dict["__init_subclass__"] = &ClassMethod{Callable: MustNewMethod("__init_subclass__", func(self Object, args Tuple, kwargs StringDict) (Object, error) {
		if len(args) != 0 || len(kwargs) != 0 {
			return nil, ExceptionNewf(TypeError, "%s.__init_subclass__() takes no arguments", self.(*Type).Name)
		}
		return None, nil
	}, 0, "Initialize a subclass; reject unconsumed class keywords."), Dict: NewStringDict()}
	TypeType.Dict["__prepare__"] = &ClassMethod{Callable: MustNewMethod("__prepare__", func(self Object, args Tuple, kwargs StringDict) (Object, error) {
		var name, bases Object
		if err := UnpackTuple(args, nil, "__prepare__", 2, 2, &name, &bases); err != nil {
			return nil, err
		}
		return NewStringDict(), nil
	}, 0, "Create a class namespace."), Dict: NewStringDict()}
	TypeType.Dict["__new__"] = &StaticMethod{Callable: MustNewMethod("__new__", func(self Object, args Tuple, kwargs StringDict) (Object, error) {
		if len(args) == 0 {
			return nil, ExceptionNewf(TypeError, "type.__new__ needs a type")
		}
		cls, ok := args[0].(*Type)
		if !ok || !cls.IsSubtype(TypeType) {
			return nil, ExceptionNewf(TypeError, "type.__new__ requires a metaclass")
		}
		return TypeNew(cls, args[1:], kwargs)
	}, 0, "Create a type."), Dict: NewStringDict()}
	TypeType.Dict["__init__"] = MustNewMethod("__init__", func(self Object, args Tuple, kwargs StringDict) (Object, error) {
		return None, typeInitDefault(self, args, kwargs)
	}, 0, "Initialize a type.")
}

func typeInitDefault(cls Object, args Tuple, kwargs StringDict) error {
	if len(args) != 1 && len(args) != 3 {
		return ExceptionNewf(TypeError, "type.__init__() takes 1 or 3 arguments")
	}
	return nil
}

// Complete type creation before the metaclass __new__ returns: initialize the
// class cell, notify descriptors from a snapshot, then run the next MRO hook.
func (t *Type) finishClass(keywords StringDict) error {
	if value, ok := t.Dict["__classcell__"]; ok {
		cell, ok := value.(*Cell)
		if !ok {
			return ExceptionNewf(TypeError, "__classcell__ must be a cell")
		}
		cell.Set(t)
		delete(t.Dict, "__classcell__")
	}
	snapshot := t.Dict.Copy()
	for name, value := range snapshot {
		_, found, err := TypeCall(value, "__set_name__", Tuple{value, t, String(name)}, nil)
		if err != nil {
			var exception *Exception
			switch info := err.(type) {
			case ExceptionInfo:
				exception, _ = info.Value.(*Exception)
			case *ExceptionInfo:
				exception, _ = info.Value.(*Exception)
			case *Exception:
				exception = info
			}
			if exception == nil {
				return err
			}
			note := String("Error calling __set_name__ on '" + value.Type().Name + "' instance '" + name + "' in '" + t.Name + "'")
			notes, ok := exception.Dict["__notes__"].(*List)
			if !ok {
				notes = NewList()
				exception.Dict["__notes__"] = notes
			}
			notes.Items = append(notes.Items, note)
			return err
		}
		_ = found
	}
	proxy := &Super{This: t, Self: t, SelfType: t}
	hook, err := proxy.M__getattribute__("__init_subclass__")
	if err != nil {
		return err
	}
	_, err = Call(hook, nil, keywords)
	return err
}
