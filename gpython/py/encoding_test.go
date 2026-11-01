package py

import "testing"

func TestDecodeSource(t *testing.T) {
	for _, test := range []struct{ data, want string }{
		{"u = 'Питон'\n", "u = 'Питон'\n"},
		{"\xef\xbb\xbfu = 1\n", "u = 1\n"},
		{"# coding: Latin-1\nu = '\xc7'\n", "# coding: Latin-1\nu = 'Ç'\n"},
		{"#!/usr/bin/python\n# coding=iso-8859-1\nu = '\xe9'\n", "#!/usr/bin/python\n# coding=iso-8859-1\nu = 'é'\n"},
	} {
		got, err := DecodeSource([]byte(test.data), "source.py")
		if err != nil || got != test.want {
			t.Errorf("DecodeSource(%q) = %q, %v", test.data, got, err)
		}
	}
	for _, data := range []string{"u = '\xff'", "# coding: nonexistent\nu = 1", "\xef\xbb\xbf# coding: latin-1\nu = 1", "# coding: ascii\nu = '\xe9'"} {
		if _, err := DecodeSource([]byte(data), "bad.py"); !IsException(SyntaxError, err) {
			t.Errorf("invalid source %q returned %v", data, err)
		}
	}
	_, err := DecodeSource([]byte("# comment\nu = '\xff'"), "bad.py")
	exception := err.(*Exception)
	if exception.Dict["lineno"] != Int(2) || exception.Dict["offset"] != Int(6) {
		t.Errorf("invalid UTF-8 location = %v", exception.Dict)
	}
}

func TestEncodeString(t *testing.T) {
	for _, test := range []struct{ text, codec, handler, want string }{
		{"Питон", "utf-8", "strict", "Питон"}, {"Ç", "Latin-1", "strict", "\xc7"},
		{"a蟒b", "ascii", "replace", "a?b"}, {"a蟒b", "latin1", "ignore", "ab"},
	} {
		got, err := EncodeString(String(test.text), test.codec, test.handler)
		if err != nil || string(got) != test.want {
			t.Errorf("encode(%q,%s) = %q, %v", test.text, test.codec, got, err)
		}
	}
	if _, err := EncodeString("蟒", "ascii", "strict"); !IsException(UnicodeEncodeError, err) {
		t.Errorf("unencodable character returned %v", err)
	}
	if _, err := EncodeString("x", "unknown", "strict"); !IsException(LookupError, err) {
		t.Errorf("unknown encoding returned %v", err)
	}
	if _, err := EncodeString("蟒", "ascii", "unknown"); !IsException(LookupError, err) {
		t.Errorf("unknown error handler returned %v", err)
	}
}
