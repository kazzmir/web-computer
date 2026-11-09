// Package xml provides XML 1.0 name and character validation utilities.
// Parser, DOM, SAX, and etree APIs are not implemented by this module.
package xml

import "github.com/kazzmir/web-computer/gpython/py"

func nameStart(c rune) bool {
	return c == ':' || c == '_' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' ||
		c >= 0xc0 && c <= 0xd6 || c >= 0xd8 && c <= 0xf6 || c >= 0xf8 && c <= 0x2ff ||
		c >= 0x370 && c <= 0x37d || c >= 0x37f && c <= 0x1fff || c >= 0x200c && c <= 0x200d ||
		c >= 0x2070 && c <= 0x218f || c >= 0x2c00 && c <= 0x2fef || c >= 0x3001 && c <= 0xd7ff ||
		c >= 0xf900 && c <= 0xfdcf || c >= 0xfdf0 && c <= 0xfffd || c >= 0x10000 && c <= 0xeffff
}
func nameChar(c rune) bool {
	return nameStart(c) || c == '-' || c == '.' || c >= '0' && c <= '9' || c == 0xb7 || c >= 0x300 && c <= 0x36f || c >= 0x203f && c <= 0x2040
}
func textChar(c rune) bool {
	return c == 9 || c == 10 || c == 13 || c >= 0x20 && c <= 0xd7ff || c >= 0xe000 && c <= 0xfffd || c >= 0x10000 && c <= 0x10ffff
}
func init() {
	py.RegisterModule(&py.ModuleImpl{
		Info: py.ModuleInfo{Name: "xml", Doc: "XML 1.0 name/text validation (no XML parser implementation)"},
		Methods: []*py.Method{
			py.MustNewMethod("is_valid_name", func(self, value py.Object) (py.Object, error) {
				s, ok := value.(py.String)
				if !ok {
					return nil, py.ExceptionNewf(py.TypeError, "name must be a string")
				}
				points := py.StringCodePoints(s)
				if len(points) == 0 || !nameStart(points[0]) {
					return py.False, nil
				}
				for _, p := range points[1:] {
					if !nameChar(p) {
						return py.False, nil
					}
				}
				return py.True, nil
			}, 0, "Check an XML 1.0 name."),
			py.MustNewMethod("is_valid_text", func(self, value py.Object) (py.Object, error) {
				s, ok := value.(py.String)
				if !ok {
					return nil, py.ExceptionNewf(py.TypeError, "text must be a string")
				}
				for _, p := range py.StringCodePoints(s) {
					if !textChar(p) {
						return py.False, nil
					}
				}
				return py.True, nil
			}, 0, "Check XML 1.0 character-data code points."),
		},
	})
}
