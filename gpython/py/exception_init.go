package py

func init() {
	BaseException.Dict["__init__"] = MustNewMethod("__init__", func(self Object, args Tuple, kwargs StringDict) (Object, error) {
		exception, ok := self.(*Exception)
		if !ok {
			if len(args) == 0 {
				return nil, ExceptionNewf(TypeError, "BaseException.__init__ requires an exception instance")
			}
			exception, ok = args[0].(*Exception)
			args = args[1:]
			if !ok {
				return nil, ExceptionNewf(TypeError, "BaseException.__init__ requires an exception instance")
			}
		}
		if len(kwargs) > 0 {
			return nil, ExceptionNewf(TypeError, "exception initialization takes no keyword arguments")
		}
		exception.Args = args.Copy()
		return None, nil
	}, 0, "Initialize the exception arguments.")
}
