package py

func init() {
	StringType.Dict["splitlines"] = MustNewMethod("splitlines", stringSplitLines, 0, "Split at Unicode line boundaries; optionally retain line endings.")
}
func stringSplitLines(self Object, args Tuple, kwargs StringDict) (Object, error) {
	var keep Object = Int(0)
	if err := ParseTupleAndKeywords(args, kwargs, "|i:splitlines", []string{"keepends"}, &keep); err != nil {
		return nil, err
	}
	text := string(self.(String))
	result := NewList()
	start := 0
	for i := 0; i < len(text); {
		point, size := DecodeCodePoint(text[i:])
		end := i + size
		boundary := point == '\n' || point == '\r' || point == '\v' || point == '\f' || point == 0x1c || point == 0x1d || point == 0x1e || point == 0x85 || point == 0x2028 || point == 0x2029
		if !boundary {
			i = end
			continue
		}
		if point == '\r' && end < len(text) && text[end] == '\n' {
			end++
		}
		stop := i
		if keep.(Int) != 0 {
			stop = end
		}
		result.Items = append(result.Items, String(text[start:stop]))
		start = end
		i = end
	}
	if start < len(text) {
		result.Items = append(result.Items, String(text[start:]))
	}
	return result, nil
}
