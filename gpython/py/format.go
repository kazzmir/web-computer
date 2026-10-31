package py

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Ascii implements the !a conversion: repr(), with non-ASCII code points escaped.
func Ascii(value Object) (Object, error) {
	text, err := ReprAsString(value)
	if err != nil {
		return nil, err
	}
	var out strings.Builder
	for _, r := range text {
		switch {
		case r < 128:
			out.WriteRune(r)
		case r <= 255:
			fmt.Fprintf(&out, "\\x%02x", r)
		case r <= 65535:
			fmt.Fprintf(&out, "\\u%04x", r)
		default:
			fmt.Fprintf(&out, "\\U%08x", r)
		}
	}
	return String(out.String()), nil
}

// Format is shared by FORMAT_VALUE and the format() builtin. Special methods
// are looked up on the type, not the instance, just like other Python protocols.
func Format(value Object, spec String) (Object, error) {
	var result Object
	var err error
	if formatter, ok := value.(I__format__); ok {
		result, err = formatter.M__format__(spec)
	} else if custom, ok, customErr := TypeCall1(value, "__format__", spec); ok {
		result, err = custom, customErr
	} else {
		if spec == "" {
			result, err = Str(value)
		} else {
			result, err = formatBuiltin(value, string(spec))
		}
	}
	if err != nil {
		return nil, err
	}
	if _, ok := result.(String); !ok {
		return nil, ExceptionNewf(TypeError, "__format__ must return a str, not %s", result.Type().Name)
	}
	return result, nil
}

type formatSpec struct {
	fill      rune
	align     rune
	sign      rune
	alternate bool
	zero      bool
	width     int
	precision int
	grouping  rune
	kind      rune
}

func parseFormatSpec(text string) (formatSpec, error) {
	s := formatSpec{fill: ' ', precision: -1}
	runes := []rune(text)
	i := 0
	align := func(r rune) bool { return strings.ContainsRune("<>=^", r) }
	if len(runes) > 1 && align(runes[1]) {
		s.fill = runes[0]
		s.align = runes[1]
		i = 2
	} else if len(runes) > 0 && align(runes[0]) {
		s.align = runes[0]
		i++
	}
	if i < len(runes) && strings.ContainsRune("+- ", runes[i]) {
		s.sign = runes[i]
		i++
	}
	if i < len(runes) && runes[i] == '#' {
		s.alternate = true
		i++
	}
	if i < len(runes) && runes[i] == '0' {
		s.zero = true
		i++
	}
	number := func() (int, error) {
		start := i
		for i < len(runes) && runes[i] >= '0' && runes[i] <= '9' {
			i++
		}
		if start == i {
			return 0, nil
		}
		n, e := strconv.Atoi(string(runes[start:i]))
		if e != nil || n > 10000000 {
			return 0, ExceptionNewf(ValueError, "format width or precision too large")
		}
		return n, nil
	}
	var err error
	s.width, err = number()
	if err != nil {
		return s, err
	}
	if i < len(runes) && (runes[i] == ',' || runes[i] == '_') {
		s.grouping = runes[i]
		i++
	}
	if i < len(runes) && runes[i] == '.' {
		i++
		if i == len(runes) || runes[i] < '0' || runes[i] > '9' {
			return s, ExceptionNewf(ValueError, "Format specifier missing precision")
		}
		s.precision, err = number()
		if err != nil {
			return s, err
		}
	}
	if i < len(runes) {
		s.kind = runes[i]
		i++
	}
	if i != len(runes) {
		return s, ExceptionNewf(ValueError, "Invalid format specifier '%s'", text)
	}
	return s, nil
}

func (s formatSpec) pad(text, prefix string, numeric bool) String {
	alignment := s.align
	fill := s.fill
	if alignment == 0 {
		if numeric {
			alignment = '>'
		} else {
			alignment = '<'
		}
	}
	if s.zero && s.fill == ' ' {
		fill = '0'
		if s.align == 0 && numeric {
			alignment = '='
		}
	}
	n := s.width - utf8.RuneCountInString(text) - utf8.RuneCountInString(prefix)
	if n <= 0 {
		return String(prefix + text)
	}
	padding := strings.Repeat(string(fill), n)
	switch alignment {
	case '<':
		return String(prefix + text + padding)
	case '^':
		left := n / 2
		return String(strings.Repeat(string(fill), left) + prefix + text + strings.Repeat(string(fill), n-left))
	case '=':
		return String(prefix + padding + text)
	default:
		return String(padding + prefix + text)
	}
}

func (s formatSpec) prefix(negative bool) string {
	if negative {
		return "-"
	}
	if s.sign == '+' {
		return "+"
	}
	if s.sign == ' ' {
		return " "
	}
	return ""
}

func groupDigits(text string, separator rune, size int, floating bool) string {
	if separator == 0 {
		return text
	}
	end := -1
	if floating {
		end = strings.IndexAny(text, ".eE%")
	}
	if end < 0 {
		end = len(text)
	}
	var out strings.Builder
	for i := 0; i < end; i++ {
		if i > 0 && (end-i)%size == 0 {
			out.WriteRune(separator)
		}
		out.WriteByte(text[i])
	}
	out.WriteString(text[end:])
	return out.String()
}

// Grouping includes sign-aware zero padding: 1234 with 010,d becomes
// 00,001,234, not 000001,234. Hexadecimal e/E are digits, not exponents.
func (s formatSpec) group(text, prefix string, size int, floating bool) string {
	if s.grouping == 0 {
		return text
	}
	if floating && (strings.HasPrefix(text, "inf") || strings.HasPrefix(text, "nan") || strings.HasPrefix(text, "INF") || strings.HasPrefix(text, "NAN")) {
		return text
	}
	zeroPad := s.zero && s.fill == ' ' && (s.align == 0 || s.align == '=') || s.fill == '0' && s.align == '='
	if zeroPad {
		end := len(text)
		if floating {
			if i := strings.IndexAny(text, ".eE%"); i >= 0 {
				end = i
			}
		}
		target := s.width - utf8.RuneCountInString(prefix) - (len(text) - end)
		low, high := end, target
		if high < low {
			high = low
		}
		for low < high {
			mid := (low + high) / 2
			if mid+(mid-1)/size < target {
				low = mid + 1
			} else {
				high = mid
			}
		}
		text = strings.Repeat("0", low-end) + text
	}
	return groupDigits(text, s.grouping, size, floating)
}

func formatBuiltin(value Object, text string) (Object, error) {
	spec, err := parseFormatSpec(text)
	if err != nil {
		return nil, err
	}
	if str, ok := value.(String); ok {
		if spec.kind != 0 && spec.kind != 's' {
			return nil, ExceptionNewf(ValueError, "Unknown format code '%c' for object of type 'str'", spec.kind)
		}
		if spec.sign != 0 || spec.alternate || spec.grouping != 0 || spec.align == '=' {
			return nil, ExceptionNewf(ValueError, "Invalid format specifier for string")
		}
		runes := []rune(str)
		if spec.precision >= 0 && spec.precision < len(runes) {
			runes = runes[:spec.precision]
		}
		return spec.pad(string(runes), "", false), nil
	}
	var integer *big.Int
	var number float64
	isFloat := false
	switch v := value.(type) {
	case Int:
		integer = big.NewInt(int64(v))
	case *BigInt:
		integer = new(big.Int).Set((*big.Int)(v))
	case Bool:
		integer = new(big.Int)
		if v {
			integer.SetInt64(1)
		}
	case Float:
		isFloat = true
		number = float64(v)
	default:
		return nil, ExceptionNewf(TypeError, "unsupported format string passed to %s.__format__", value.Type().Name)
	}
	if isFloat || strings.ContainsRune("eEfFgG%", spec.kind) {
		if !isFloat {
			number, _ = new(big.Float).SetInt(integer).Float64()
			if math.IsInf(number, 0) {
				return nil, ExceptionNewf(OverflowError, "int too large to convert to float")
			}
		}
		return formatFloat(number, spec)
	}
	if spec.precision >= 0 {
		return nil, ExceptionNewf(ValueError, "Precision not allowed in integer format specifier")
	}
	base := 10
	prefix := ""
	groupSize := 3
	switch spec.kind {
	case 0, 'd', 'n':
		if spec.kind == 'n' && spec.grouping != 0 {
			return nil, ExceptionNewf(ValueError, "Cannot specify grouping with 'n'")
		}
	case 'b':
		base = 2
		prefix = "0b"
		groupSize = 4
	case 'o':
		base = 8
		prefix = "0o"
		groupSize = 4
	case 'x', 'X':
		base = 16
		prefix = "0x"
		groupSize = 4
	case 'c':
		if spec.sign != 0 || spec.alternate || spec.grouping != 0 {
			return nil, ExceptionNewf(ValueError, "Invalid format specifier for character")
		}
		if !integer.IsInt64() || integer.Sign() < 0 || integer.Int64() > utf8.MaxRune {
			return nil, ExceptionNewf(OverflowError, "%c arg not in range(0x110000)", 'c')
		}
		return spec.pad(string(rune(integer.Int64())), "", true), nil
	default:
		return nil, ExceptionNewf(ValueError, "Unknown format code '%c' for object of type 'int'", spec.kind)
	}
	if spec.grouping == ',' && base != 10 {
		return nil, ExceptionNewf(ValueError, "Cannot specify ',' with '%c'", spec.kind)
	}
	negative := integer.Sign() < 0
	digits := new(big.Int).Abs(integer).Text(base)
	if !spec.alternate {
		prefix = ""
	}
	if spec.kind == 'X' {
		digits = strings.ToUpper(digits)
		prefix = strings.ToUpper(prefix)
	}
	prefix = spec.prefix(negative) + prefix
	digits = spec.group(digits, prefix, groupSize, false)
	return spec.pad(digits, prefix, true), nil
}

func formatFloat(value float64, spec formatSpec) (Object, error) {
	kind := spec.kind
	precision := spec.precision
	if kind == 0 {
		kind = 'g'
		if precision < 0 {
			precision = -1
		}
	} else if precision < 0 {
		precision = 6
	}
	if kind == 'n' {
		if spec.grouping != 0 {
			return nil, ExceptionNewf(ValueError, "Cannot specify grouping with 'n'")
		}
		kind = 'g'
	}
	if !strings.ContainsRune("eEfFgG%", kind) {
		return nil, ExceptionNewf(ValueError, "Unknown format code '%c' for object of type 'float'", kind)
	}
	negative := math.Signbit(value)
	value = math.Abs(value)
	suffix := ""
	if kind == '%' {
		value *= 100
		kind = 'f'
		suffix = "%"
	}
	lower := byte(kind)
	if lower >= 'A' && lower <= 'Z' {
		lower += 'a' - 'A'
	}
	var digits string
	if math.IsInf(value, 0) {
		digits = "inf"
	} else if math.IsNaN(value) {
		digits = "nan"
	} else {
		if spec.alternate && precision >= 0 {
			// Go's # formatting retains trailing zeroes for g/G and the decimal point
			// at zero precision, matching Python's alternate numeric form.
			digits = fmt.Sprintf("%#.*"+string(kind), precision, value)
		} else {
			digits = strconv.FormatFloat(value, lower, precision, 64)
		}
		if spec.kind == 0 {
			if precision >= 0 && value != 0 {
				p := precision
				if p == 0 {
					p = 1
				}
				rounded, _ := strconv.ParseFloat(digits, 64)
				if math.Log10(rounded) >= float64(p-1) && !strings.ContainsAny(digits, "eE") {
					digits = strconv.FormatFloat(value, 'e', p-1, 64)
					if spec.alternate && !strings.Contains(digits, ".") {
						digits = strings.Replace(digits, "e", ".e", 1)
					}
				}
			}
			if !strings.ContainsAny(digits, ".eE") {
				digits += ".0"
			}
		}
	}
	if kind == 'E' || kind == 'F' || kind == 'G' {
		digits = strings.ToUpper(digits)
	}
	prefix := spec.prefix(negative)
	digits = spec.group(digits+suffix, prefix, 3, true)
	return spec.pad(digits, prefix, true), nil
}
