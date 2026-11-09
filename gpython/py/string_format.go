package py

import (
	"strconv"
	"strings"
)

type stringFormatter struct {
	args         Tuple
	kwargs       StringDict
	automatic    int
	auto, manual bool
}

func (s String) Format(args Tuple, kwargs StringDict) (Object, error) {
	f := stringFormatter{args: args, kwargs: kwargs}
	text, err := f.render(string(s), 0)
	return String(text), err
}
func (f *stringFormatter) field(name string) (Object, error) {
	end := strings.IndexAny(name, ".[")
	if end < 0 {
		end = len(name)
	}
	base, tail := name[:end], name[end:]
	var value Object
	if base == "" {
		if f.manual {
			return nil, ExceptionNewf(ValueError, "cannot switch from manual field specification to automatic field numbering")
		}
		f.auto = true
		if f.automatic >= len(f.args) {
			return nil, ExceptionNewf(IndexError, "replacement index out of range")
		}
		value = f.args[f.automatic]
		f.automatic++
	} else if index, err := strconv.Atoi(base); err == nil && index >= 0 {
		if f.auto {
			return nil, ExceptionNewf(ValueError, "cannot switch from automatic field numbering to manual field specification")
		}
		f.manual = true
		if index >= len(f.args) {
			return nil, ExceptionNewf(IndexError, "replacement index out of range")
		}
		value = f.args[index]
	} else {
		var ok bool
		value, ok = f.kwargs[base]
		if !ok {
			return nil, ExceptionNewf(KeyError, "%s", base)
		}
	}
	for len(tail) > 0 {
		var err error
		switch tail[0] {
		case '.':
			rest := tail[1:]
			end := strings.IndexAny(rest, ".[")
			if end < 0 {
				end = len(rest)
			}
			if end == 0 {
				return nil, ExceptionNewf(ValueError, "empty attribute in format string")
			}
			value, err = GetAttrString(value, rest[:end])
			tail = rest[end:]
		case '[':
			end := strings.IndexByte(tail, ']')
			if end < 0 {
				return nil, ExceptionNewf(ValueError, "missing ']' in format string")
			}
			key := tail[1:end]
			var object Object = String(key)
			if index, e := strconv.Atoi(key); e == nil && index >= 0 {
				object = Int(index)
			}
			value, err = GetItem(value, object)
			tail = tail[end+1:]
		default:
			return nil, ExceptionNewf(ValueError, "invalid field access in format string")
		}
		if err != nil {
			return nil, err
		}
	}
	return value, nil
}
func (f *stringFormatter) replacement(field string, depth int) (string, error) {
	split := len(field)
	brackets := 0
	for i, c := range field {
		if c == '[' {
			brackets++
		}
		if c == ']' {
			brackets--
		}
		if brackets == 0 && (c == '!' || c == ':') {
			split = i
			break
		}
	}
	value, err := f.field(field[:split])
	if err != nil {
		return "", err
	}
	rest := field[split:]
	if strings.HasPrefix(rest, "!") {
		if len(rest) < 2 {
			return "", ExceptionNewf(ValueError, "missing conversion specifier")
		}
		switch rest[1] {
		case 'r':
			value, err = Repr(value)
		case 's':
			value, err = Str(value)
		case 'a':
			value, err = Ascii(value)
		default:
			return "", ExceptionNewf(ValueError, "unknown conversion specifier %c", rest[1])
		}
		if err != nil {
			return "", err
		}
		if _, ok := value.(String); !ok {
			return "", ExceptionNewf(TypeError, "format conversion must return str")
		}
		rest = rest[2:]
	}
	spec := ""
	if len(rest) > 0 {
		if rest[0] != ':' {
			return "", ExceptionNewf(ValueError, "expected ':' after conversion specifier")
		}
		spec, err = f.render(rest[1:], depth+1)
		if err != nil {
			return "", err
		}
	}
	result, err := Format(value, String(spec))
	if err != nil {
		return "", err
	}
	return string(result.(String)), nil
}
func (f *stringFormatter) render(text string, depth int) (string, error) {
	if depth > 2 {
		return "", ExceptionNewf(ValueError, "max string format recursion exceeded")
	}
	var out strings.Builder
	for i := 0; i < len(text); {
		c := text[i]
		if c != '{' && c != '}' {
			out.WriteByte(c)
			i++
			continue
		}
		if i+1 < len(text) && text[i+1] == c {
			out.WriteByte(c)
			i += 2
			continue
		}
		if c == '}' {
			return "", ExceptionNewf(ValueError, "single '}' encountered in format string")
		}
		start := i + 1
		i++
		nest := 0
		for i < len(text) {
			if text[i] == '{' {
				nest++
			}
			if text[i] == '}' {
				if nest == 0 {
					break
				}
				nest--
			}
			i++
		}
		if i == len(text) {
			return "", ExceptionNewf(ValueError, "expected '}' before end of string")
		}
		value, err := f.replacement(text[start:i], depth)
		if err != nil {
			return "", err
		}
		out.WriteString(value)
		i++
	}
	return out.String(), nil
}
