package xml

import "testing"

func TestXMLCharacterBoundaries(t *testing.T) {
	for _, test := range []struct {
		point rune
		valid bool
	}{
		{0, false}, {9, true}, {10, true}, {13, true}, {0x1f, false}, {0x20, true},
		{0xd7ff, true}, {0xd800, false}, {0xdfff, false}, {0xe000, true}, {0xfffd, true},
		{0xfffe, false}, {0xffff, false}, {0x10000, true}, {0x10ffff, true}, {0x110000, false},
	} {
		if got := textChar(test.point); got != test.valid {
			t.Errorf("text U+%04X = %v", test.point, got)
		}
	}
	for _, test := range []struct {
		point               rune
		start, continuation bool
	}{
		{'A', true, true}, {'0', false, true}, {'-', false, true}, {0xb7, false, true},
		{0x300, false, true}, {0x370, true, true}, {0x37e, false, false},
		{0xd800, false, false}, {0xeffff, true, true}, {0xf0000, false, false},
	} {
		if nameStart(test.point) != test.start || nameChar(test.point) != test.continuation {
			t.Errorf("name U+%04X", test.point)
		}
	}
}
