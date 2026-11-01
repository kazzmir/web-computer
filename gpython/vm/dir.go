package vm

import (
	"reflect"
	"sort"
	"strings"

	"github.com/kazzmir/web-computer/gpython/py"
)

func builtinDir(frame *py.Frame, args py.Tuple, kwargs py.StringDict) (py.Object, error) {
	var object py.Object
	if err := py.UnpackTuple(args, kwargs, "dir", 0, 1, &object); err != nil {
		return nil, err
	}
	names := map[string]bool{}
	add := func(dict py.StringDict) {
		for name := range dict {
			names[name] = true
		}
	}
	if object == nil {
		frame.FastToLocals()
		add(frame.Locals)
	} else {
		if result, found, err := py.TypeCall0(object, "__dir__"); found {
			if err != nil {
				return nil, err
			}
			list, err := py.SequenceList(result)
			if err != nil {
				return nil, err
			}
			method, err := py.GetAttrString(list, "sort")
			if err != nil {
				return nil, err
			}
			if _, err := py.Call(method, nil, nil); err != nil {
				return nil, err
			}
			return list, nil
		}
		if dict, ok := object.(py.IGetDict); ok {
			add(dict.GetDict())
		}
		class := object.Type()
		if own, ok := object.(*py.Type); ok && len(own.Mro) > 0 {
			class = own
		}
		for _, base := range class.Mro {
			add(base.(*py.Type).Dict)
		}
		typ := reflect.TypeOf(object)
		for i := 0; i < typ.NumMethod(); i++ {
			name := typ.Method(i).Name
			if strings.HasPrefix(name, "M__") && strings.HasSuffix(name, "__") {
				names[name[1:]] = true
			}
		}
	}
	sorted := make([]string, 0, len(names))
	for name := range names {
		sorted = append(sorted, name)
	}
	sort.Strings(sorted)
	return py.NewListFromStrings(sorted), nil
}
