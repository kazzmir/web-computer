package py

import "unicode/utf8"

// DecodeCodePoint extends UTF-8 with the three-byte surrogate encodings used
// internally for Python str. Go's normal rune iteration would replace these
// valid Python code points with U+FFFD, silently changing the string.
func DecodeCodePoint(text string) (rune, int) {
	if len(text) >= 3 && text[0] == 0xed && text[1] >= 0xa0 && text[1] <= 0xbf && text[2] >= 0x80 && text[2] <= 0xbf {
		return rune(text[0]&15)<<12 | rune(text[1]&63)<<6 | rune(text[2]&63), 3
	}
	return utf8.DecodeRuneInString(text)
}

func StringFromCodePoint(r rune) String {
	if r >= 0xd800 && r <= 0xdfff {
		return String([]byte{0xe0 | byte(r>>12), 0x80 | byte(r>>6&63), 0x80 | byte(r&63)})
	}
	return String(string(r))
}

func StringCodePoints(text String) []rune {
	result := make([]rune, 0, len(text))
	for len(text) > 0 {
		r, size := DecodeCodePoint(string(text))
		result = append(result, r)
		text = text[size:]
	}
	return result
}

func StringFromCodePoints(points []rune) String {
	result := make([]byte, 0, len(points))
	for _, r := range points {
		result = append(result, []byte(StringFromCodePoint(r))...)
	}
	return String(result)
}
