package py

func init() {
	StringDictType.Dict["update"] = MustNewMethod("update", func(self Object, args Tuple, kwargs StringDict) (Object, error) {
		source, err := DictNew(StringDictType, args, kwargs)
		if err != nil {
			return nil, err
		}
		for _, entry := range dictEntries(source) {
			if _, err := SetItem(self, entry[0], entry[1]); err != nil {
				return nil, err
			}
		}
		return None, nil
	}, 0, "Update keys from a mapping or iterable of pairs and keyword arguments.")
	StringDictType.Dict["copy"] = MustNewMethod("copy", func(self Object, args Tuple) (Object, error) {
		if err := UnpackTuple(args, nil, "copy", 0, 0); err != nil {
			return nil, err
		}
		return DictNew(StringDictType, Tuple{self}, nil)
	}, 0, "Return a shallow copy of the dictionary.")
}
