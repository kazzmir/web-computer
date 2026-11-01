// Package html provides HTML escaping and HTML5 character-reference decoding.
package html

import (
	stdhtml "html"
	"regexp"
	"strconv"
	"strings"

	"github.com/kazzmir/web-computer/gpython/py"
)

var references = regexp.MustCompile(`&(#[0-9]+;?|#[xX][0-9a-fA-F]+;?|[^\t\n\f <&#;]{1,32};?)`)

func unescape(text string) string {
	return references.ReplaceAllStringFunc(text, func(reference string) string {
		if len(reference) < 2 || reference[1] != '#' {
			return stdhtml.UnescapeString(reference)
		}
		digits := strings.TrimSuffix(reference[2:], ";")
		base := 10
		if strings.HasPrefix(digits, "x") || strings.HasPrefix(digits, "X") {
			base = 16
			digits = digits[1:]
		}
		cp, err := strconv.ParseUint(digits, base, 64)
		if err != nil || cp > 0x10ffff {
			return "\ufffd"
		}
		if cp == 0 || (cp >= 0xd800 && cp <= 0xdfff) {
			return "\ufffd"
		}
		// Python follows HTML5's invalid-codepoint removal rules rather than
		// retaining control/noncharacter code points as Go's html package does.
		if cp < 0x20 && cp != 9 && cp != 10 && cp != 12 && cp != 13 || cp == 0x7f || cp >= 0xfdd0 && cp <= 0xfdef || cp&0xffff == 0xfffe || cp&0xffff == 0xffff {
			return ""
		}
		if cp >= 0x80 && cp <= 0x9f {
			return stdhtml.UnescapeString("&#" + strconv.FormatUint(cp, 10) + ";")
		}
		return string(rune(cp))
	})
}

func init() {
	py.RegisterModule(&py.ModuleImpl{
		Info: py.ModuleInfo{Name: "html", Doc: "HTML escaping and HTML5 character references"},
		Methods: []*py.Method{
			py.MustNewMethod("escape", func(self py.Object, args py.Tuple, kwargs py.StringDict) (py.Object, error) {
				var text py.Object
				var quote py.Object = py.True
				if err := py.ParseTupleAndKeywords(args, kwargs, "O|O:escape", []string{"s", "quote"}, &text, &quote); err != nil {
					return nil, err
				}
				s, ok := text.(py.String)
				if !ok {
					return nil, py.ExceptionNewf(py.TypeError, "escape requires a string")
				}
				value := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(string(s))
				quoted, err := py.ObjectIsTrue(quote)
				if err != nil {
					return nil, err
				}
				if quoted {
					value = strings.NewReplacer("\"", "&quot;", "'", "&#x27;").Replace(value)
				}
				return py.String(value), nil
			}, 0, "Escape HTML special characters, optionally including quotes."),
			py.MustNewMethod("unescape", func(self py.Object, args py.Tuple, kwargs py.StringDict) (py.Object, error) {
				var text py.Object
				if err := py.ParseTupleAndKeywords(args, kwargs, "O:unescape", []string{"s"}, &text); err != nil {
					return nil, err
				}
				s, ok := text.(py.String)
				if !ok {
					return nil, py.ExceptionNewf(py.TypeError, "unescape requires a string")
				}
				return py.String(unescape(string(s))), nil
			}, 0, "Decode HTML5 named and numeric character references."),
		},
	})
}
