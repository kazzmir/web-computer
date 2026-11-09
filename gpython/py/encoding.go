package py

import (
	"bytes"
	"regexp"
	"strings"
	"unicode/utf8"
)

func encodingName(name string) string {
	return strings.NewReplacer("-", "", "_", "", " ", "").Replace(strings.ToLower(name))
}

// EncodeString supplies the basic codecs needed by Python source and text.
// Other codecs fail explicitly rather than silently treating bytes as UTF-8.
func EncodeString(text String, encoding, errors string) (Bytes, error) {
	name := encodingName(encoding)
	if name == "utf8" {
		if utf8.ValidString(string(text)) || errors == "surrogatepass" {
			return Bytes([]byte(text)), nil
		}
		var result Bytes
		for _, point := range StringCodePoints(text) {
			if point < 0xd800 || point > 0xdfff {
				result = append(result, []byte(string(point))...)
				continue
			}
			switch errors {
			case "ignore":
			case "replace":
				result = append(result, '?')
			case "strict":
				return nil, ExceptionNewf(UnicodeEncodeError, "'utf-8' codec can't encode character %U: surrogates not allowed", point)
			default:
				return nil, ExceptionNewf(LookupError, "unknown error handler name: %s", errors)
			}
		}
		return result, nil
	}
	limit := rune(255)
	switch name {
	case "latin1", "iso88591", "iso8859", "l1":
	case "ascii", "usascii":
		limit = 127
	default:
		return nil, ExceptionNewf(LookupError, "unknown encoding: %s", encoding)
	}
	out := make(Bytes, 0, len(text))
	for _, r := range StringCodePoints(text) {
		if r <= limit {
			out = append(out, byte(r))
			continue
		}
		switch errors {
		case "ignore":
			continue
		case "replace":
			out = append(out, '?')
		case "strict":
			return nil, ExceptionNewf(UnicodeEncodeError, "'%s' codec can't encode character %U", encoding, r)
		default:
			return nil, ExceptionNewf(LookupError, "unknown error handler name: %s", errors)
		}
	}
	return out, nil
}

func init() {
	StringType.Dict["encode"] = MustNewMethod("encode", func(self Object, args Tuple, kwargs StringDict) (Object, error) {
		var encoding Object = String("utf-8")
		var errors Object = String("strict")
		if err := ParseTupleAndKeywords(args, kwargs, "|ss:encode", []string{"encoding", "errors"}, &encoding, &errors); err != nil {
			return nil, err
		}
		return EncodeString(self.(String), string(encoding.(String)), string(errors.(String)))
	}, 0, "Encode text using UTF-8, Latin-1, or ASCII.")
}

var sourceEncoding = regexp.MustCompile(`^[\t\f ]*#.*?coding[:=][\t ]*([-_.a-zA-Z0-9]+)`)

// DecodeSource decodes file/bytes source per PEP 263/3120. Already-decoded
// Python str source must bypass this: encoding comments do not reinterpret it.
func DecodeSource(data []byte, filename string) (string, error) {
	bom := bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf})
	if bom {
		data = data[3:]
	}
	encoding := "utf-8"
	lines := bytes.SplitN(data, []byte{'\n'}, 3)
	for i, line := range lines {
		if i == 2 {
			break
		}
		if cookie := sourceEncoding.FindSubmatch(line); cookie != nil {
			encoding = string(cookie[1])
			break
		}
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) > 0 && trimmed[0] != '#' {
			break
		}
	}
	syntaxAt := func(message string, offset int) (string, error) {
		start := bytes.LastIndexByte(data[:offset], '\n') + 1
		end := bytes.IndexByte(data[offset:], '\n')
		if end < 0 {
			end = len(data)
		} else {
			end += offset
		}
		line := bytes.Count(data[:start], []byte{'\n'}) + 1
		return "", MakeSyntaxError(ExceptionNewf(SyntaxError, "%s", message), filename, line, offset-start, string(data[start:end]))
	}
	syntax := func(message string) (string, error) { return syntaxAt(message, 0) }
	name := encodingName(encoding)
	if bom && name != "utf8" {
		return syntax("encoding problem: UTF-8 BOM conflicts with coding declaration")
	}
	switch name {
	case "utf8":
		if !utf8.Valid(data) {
			for offset := 0; offset < len(data); {
				_, size := utf8.DecodeRune(data[offset:])
				if size == 1 && data[offset] >= utf8.RuneSelf {
					return syntaxAt("source is not valid utf-8", offset)
				}
				offset += size
			}
		}
		return string(data), nil
	case "latin1", "iso88591", "iso8859", "l1":
		runes := make([]rune, len(data))
		for i, b := range data {
			runes[i] = rune(b)
		}
		return string(runes), nil
	case "ascii", "usascii":
		for i, b := range data {
			if b > 127 {
				return syntaxAt("source is not valid ascii", i)
			}
		}
		return string(data), nil
	default:
		return syntax("unknown encoding: " + encoding)
	}
}
