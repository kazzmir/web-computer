package py

import (
	"math"
	"strings"
)

// GenericDict backs dictionaries containing non-string keys. Namespace maps
// remain StringDicts. Lookup is linear for now, using Python equality rather
// than Go map-key equality (so e.g. 1, True and 1.0 share a key).
type GenericDict struct{ entries []Tuple }

func NewGenericDict() *GenericDict { return &GenericDict{} }

// DictBuilderSet may promote a not-yet-published literal/constructor result.
// It must not be used for mutation of dictionaries that already have aliases.
func DictBuilderSet(dict, key, value Object) (Object, error) {
	if strings, ok := dict.(StringDict); ok {
		if _, isString := key.(String); !isString {
			promoted := NewGenericDict()
			for k, v := range strings {
				promoted.entries = append(promoted.entries, Tuple{String(k), v})
			}
			dict = promoted
		}
	}
	_, err := SetItem(dict, key, value)
	return dict, err
}
func (d *GenericDict) Type() *Type { return StringDictType }
func dictEntries(obj Object) []Tuple {
	if d, ok := obj.(*GenericDict); ok {
		return d.entries
	}
	d := obj.(StringDict)
	entries := make([]Tuple, 0, len(d))
	for k, v := range d {
		entries = append(entries, Tuple{String(k), v})
	}
	return entries
}
func dictKeyCheck(key Object) error {
	switch k := key.(type) {
	case Float:
		if math.IsNaN(float64(k)) {
			return ExceptionNewf(NotImplementedError, "NaN dictionary keys are not supported")
		}
		return nil
	case Complex:
		if math.IsNaN(real(complex128(k))) || math.IsNaN(imag(complex128(k))) {
			return ExceptionNewf(NotImplementedError, "NaN dictionary keys are not supported")
		}
		return nil
	case String, Bytes, Int, *BigInt, Bool, NoneType:
		return nil
	case Tuple:
		for _, item := range k {
			if err := dictKeyCheck(item); err != nil {
				return err
			}
		}
		return nil
	default:
		return ExceptionNewf(TypeError, "unsupported dictionary key type: '%s'", key.Type().Name)
	}
}
func (d *GenericDict) find(key Object) (int, error) {
	if err := dictKeyCheck(key); err != nil {
		return -1, err
	}
	for i, entry := range d.entries {
		equal, err := Eq(entry[0], key)
		if err != nil {
			return -1, err
		}
		truth, err := ObjectIsTrue(equal)
		if err != nil {
			return -1, err
		}
		if truth {
			return i, nil
		}
	}
	return -1, nil
}
func (d *GenericDict) M__setitem__(key, value Object) (Object, error) {
	i, err := d.find(key)
	if err != nil {
		return nil, err
	}
	if i < 0 {
		d.entries = append(d.entries, Tuple{key, value})
	} else {
		d.entries[i][1] = value
	}
	return None, nil
}
func (d *GenericDict) M__getitem__(key Object) (Object, error) {
	i, err := d.find(key)
	if err != nil {
		return nil, err
	}
	if i < 0 {
		return nil, ExceptionNewf(KeyError, "%v", key)
	}
	return d.entries[i][1], nil
}
func (d *GenericDict) M__delitem__(key Object) (Object, error) {
	i, err := d.find(key)
	if err != nil {
		return nil, err
	}
	if i < 0 {
		return nil, ExceptionNewf(KeyError, "%v", key)
	}
	d.entries = append(d.entries[:i], d.entries[i+1:]...)
	return None, nil
}
func (d *GenericDict) M__contains__(key Object) (Object, error) {
	i, err := d.find(key)
	return Bool(i >= 0), err
}
func (d *GenericDict) M__len__() (Object, error)  { return Int(len(d.entries)), nil }
func (d *GenericDict) M__bool__() (Object, error) { return Bool(len(d.entries) > 0), nil }
func (d *GenericDict) M__iter__() (Object, error) {
	keys := make(Tuple, 0, len(d.entries))
	for _, e := range d.entries {
		keys = append(keys, e[0])
	}
	return NewIterator(keys), nil
}
func (d *GenericDict) M__repr__() (Object, error) {
	parts := make([]string, 0, len(d.entries))
	for _, e := range d.entries {
		k, err := ReprAsString(e[0])
		if err != nil {
			return nil, err
		}
		v, err := ReprAsString(e[1])
		if err != nil {
			return nil, err
		}
		parts = append(parts, k+": "+v)
	}
	return String("{" + strings.Join(parts, ", ") + "}"), nil
}
func (d *GenericDict) M__str__() (Object, error) { return d.M__repr__() }
func (d *GenericDict) M__eq__(other Object) (Object, error) {
	switch other.(type) {
	case StringDict, *GenericDict:
	default:
		return NotImplemented, nil
	}
	entries := dictEntries(other)
	if len(entries) != len(d.entries) {
		return False, nil
	}
	for _, e := range d.entries {
		value, err := GetItem(other, e[0])
		if err != nil {
			if IsException(KeyError, err) {
				return False, nil
			}
			return nil, err
		}
		equal, err := Eq(e[1], value)
		if err != nil {
			return nil, err
		}
		truth, err := ObjectIsTrue(equal)
		if err != nil {
			return nil, err
		}
		if !truth {
			return False, nil
		}
	}
	return True, nil
}
func (d *GenericDict) M__ne__(other Object) (Object, error) {
	value, err := d.M__eq__(other)
	if err != nil || value == NotImplemented {
		return value, err
	}
	return Bool(value == False), nil
}
