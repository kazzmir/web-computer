package py

func init() {
	ListType.Dict["insert"] = MustNewMethod("insert", func(self Object, args Tuple) (Object, error) {
		var position, value Object
		if err := UnpackTuple(args, nil, "insert", 2, 2, &position, &value); err != nil {
			return nil, err
		}
		index, err := IndexInt(position)
		if err != nil {
			return nil, err
		}
		list := self.(*List)
		if index < 0 {
			index += len(list.Items)
			if index < 0 {
				index = 0
			}
		}
		if index > len(list.Items) {
			index = len(list.Items)
		}
		list.Items = append(list.Items, nil)
		copy(list.Items[index+1:], list.Items[index:])
		list.Items[index] = value
		return None, nil
	}, 0, "Insert an item before the given index.")
}
