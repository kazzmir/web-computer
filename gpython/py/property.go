// Copyright 2018 The go-python Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Property object

package py

// A python Property object
type Property struct {
	Fget func(self Object) (Object, error)
	Fset func(self, value Object) error
	Fdel func(self Object) error
	Doc  string
}

var PropertyType = ObjectType.NewType("property", "property(fget=None, fset=None, fdel=None, doc=None)", PropertyNew, nil)

func PropertyNew(metatype *Type, args Tuple, kwargs StringDict) (Object, error) {
	var get, set, del, doc Object = None, None, None, None
	if err := ParseTupleAndKeywords(args, kwargs, "|OOOO:property", []string{"fget", "fset", "fdel", "doc"}, &get, &set, &del, &doc); err != nil {
		return nil, err
	}
	p := &Property{}
	if get != None {
		p.Fget = func(self Object) (Object, error) { return Call(get, Tuple{self}, nil) }
	}
	if set != None {
		p.Fset = func(self, value Object) error { _, err := Call(set, Tuple{self, value}, nil); return err }
	}
	if del != None {
		p.Fdel = func(self Object) error { _, err := Call(del, Tuple{self}, nil); return err }
	}
	if text, ok := doc.(String); ok {
		p.Doc = string(text)
	}
	return p, nil
}

// Type of this object
func (o *Property) Type() *Type {
	return PropertyType
}

func (p *Property) M__get__(instance, owner Object) (Object, error) {
	if instance == None {
		return p, nil
	}
	if p.Fget == nil {
		return nil, ExceptionNewf(AttributeError, "can't get attribute")
	}
	return p.Fget(instance)
}

func (p *Property) M__set__(instance, value Object) (Object, error) {
	if p.Fset == nil {
		return nil, ExceptionNewf(AttributeError, "can't set attribute")
	}
	return None, p.Fset(instance, value)
}

func (p *Property) M__delete__(instance Object) (Object, error) {
	if p.Fdel == nil {
		return nil, ExceptionNewf(AttributeError, "can't delete attribute")
	}
	return None, p.Fdel(instance)
}

// Interfaces
var _ I__get__ = (*Property)(nil)
var _ I__set__ = (*Property)(nil)
var _ I__delete__ = (*Property)(nil)
