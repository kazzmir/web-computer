package py

import "unicode"

func init() {
	StringType.Dict["isupper"] = MustNewMethod("isupper", func(self Object, args Tuple) (Object, error) {
		if err := UnpackTuple(args, nil, "isupper", 0, 0); err != nil {
			return nil, err
		}
		found := false
		for _, c := range StringCodePoints(self.(String)) {
			if unicode.IsLower(c) || unicode.Is(unicode.Other_Lowercase, c) || unicode.IsTitle(c) {
				return False, nil
			}
			if unicode.IsUpper(c) || unicode.Is(unicode.Other_Uppercase, c) {
				found = true
			}
		}
		return Bool(found), nil
	}, 0, "Return true if there is a cased character and every cased character is uppercase.")
}
