package py

func copyMethodMetadata(callable Object, dict StringDict) error {
	for _, name := range []string{"__module__", "__qualname__", "__name__", "__doc__", "__annotations__"} {
		value, err := GetAttrString(callable, name)
		if err != nil {
			if IsException(AttributeError, err) {
				continue
			}
			return err
		}
		dict[name] = value
	}
	return nil
}
func (c *StaticMethod) M__call__(args Tuple, kwargs StringDict) (Object, error) {
	return Call(c.Callable, args, kwargs)
}
func (c *StaticMethod) M__repr__() (Object, error) {
	text, err := ReprAsString(c.Callable)
	if err != nil {
		return nil, err
	}
	return String("<staticmethod(" + text + ")>"), nil
}
func (c *ClassMethod) M__repr__() (Object, error) {
	text, err := ReprAsString(c.Callable)
	if err != nil {
		return nil, err
	}
	return String("<classmethod(" + text + ")>"), nil
}
func init() {
	StaticMethodType.Dict["__wrapped__"] = &Property{Fget: func(self Object) (Object, error) { return self.(*StaticMethod).Callable, nil }}
	ClassMethodType.Dict["__wrapped__"] = &Property{Fget: func(self Object) (Object, error) { return self.(*ClassMethod).Callable, nil }}
}
