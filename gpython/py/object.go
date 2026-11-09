// Copyright 2018 The go-python Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Functions to operate on any object

package py

import (
	"fmt"
)

// Gets the attribute attr from object or returns nil
func ObjectGetAttr(o Object, attr string) Object {
	// FIXME
	return nil
}

// Gets the repr for an object
func ObjectRepr(o Object) Object {
	// FIXME
	return String(fmt.Sprintf("<%s %v>", o.Type().Name, o))
}

// Return whether the object is True or not
func ObjectIsTrue(o Object) (cmp bool, err error) {
	switch o {
	case True:
		return true, nil
	case False:
		return false, nil
	case None:
		return false, nil
	}

	var res Object
	boolSlot, lenSlot := false, false
	switch t := o.(type) {
	case I__bool__:
		boolSlot = true
		res, err = t.M__bool__()
	case I__len__:
		lenSlot = true
		res, err = t.M__len__()
	case *Type:
		var ok bool
		if res, ok, err = TypeCall0(o, "__bool__"); ok {
			boolSlot = true
			break
		}
		if res, ok, err = TypeCall0(o, "__len__"); ok {
			lenSlot = true
			break
		}
		_ = ok // pass static-check
	}
	if err != nil {
		return false, err
	}

	if boolSlot {
		value, ok := res.(Bool)
		if !ok {
			return false, ExceptionNewf(TypeError, "__bool__ should return bool, returned %s", res.Type().Name)
		}
		return bool(value), nil
	}
	if lenSlot {
		length, err := IndexInt(res)
		if err != nil {
			return false, err
		}
		if length < 0 {
			return false, ExceptionNewf(ValueError, "__len__() should return >= 0")
		}
		return length != 0, nil
	}
	return true, nil
}

// Return whether the object is a sequence
func ObjectIsSequence(o Object) bool {
	switch t := o.(type) {
	case I__getitem__:
		return true
	case *Type:
		if t.GetAttrOrNil("__getitem__") != nil {
			return true
		}
	}
	return false
}
