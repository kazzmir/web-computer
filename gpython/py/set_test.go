package py

import "testing"

func TestFrozenSetConstructorAndMembership(t *testing.T) {
	source := NewListFromItems([]Object{String("one"), String("two"), String("one")})
	obj, err := FrozenSetNew(FrozenSetType, Tuple{source}, nil)
	if err != nil {
		t.Fatal(err)
	}
	frozen := obj.(*FrozenSet)
	source.Items[0] = String("changed")
	if size, _ := frozen.M__len__(); size != Int(2) {
		t.Fatalf("size = %v, want 2", size)
	}
	for _, item := range []struct {
		value Object
		want  Bool
	}{{String("one"), True}, {String("changed"), False}, {None, False}} {
		got, err := frozen.M__contains__(item.value)
		if err != nil || got != item.want {
			t.Errorf("contains %v = %v, %v", item.value, got, err)
		}
	}
	again, err := FrozenSetNew(FrozenSetType, Tuple{frozen}, nil)
	if err != nil || again != frozen {
		t.Errorf("frozenset(existing) = %v, %v", again, err)
	}
	empty, err := FrozenSetNew(FrozenSetType, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if size, _ := empty.(*FrozenSet).M__len__(); size != Int(0) {
		t.Errorf("empty size = %v", size)
	}
	if _, err := FrozenSetNew(FrozenSetType, Tuple{source, source}, nil); !IsException(TypeError, err) {
		t.Errorf("extra argument returned %v", err)
	}
}
