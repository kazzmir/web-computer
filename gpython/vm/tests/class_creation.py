doc = 'PEP 487 hooks, constructor dispatch, class cells and cooperative super'
import types
calls = []
class Descriptor:
    def __set_name__(self, owner, name):
        calls.append(('descriptor', name))
        self.owner = owner
class Base:
    def __init_subclass__(cls, tag=None, **kw):
        super().__init_subclass__(**kw)
        assert cls.marker.owner is cls
        assert cls.probe() is cls
        calls.append(('subclass', tag))
class Meta(type):
    def __new__(cls, name, bases, ns, **kw):
        result = super().__new__(cls, name, bases, ns, **kw)
        assert result.marker.owner is result
        calls.append(('new', name))
        return result
    def __init__(cls, name, bases, ns, **kw):
        super().__init__(name, bases, ns)
        calls.append(('init', name))
class Child(Base, metaclass=Meta, tag='tag'):
    marker = Descriptor()
    @classmethod
    def probe(cls): return __class__
assert calls == [('descriptor', 'marker'), ('subclass', 'tag'), ('new', 'Child'), ('init', 'Child')]
assert '__classcell__' not in Child.__dict__
assert Child.__name__ == 'Child'
assert Child.__qualname__ == 'Child'
class Root:
    def method(self): return ['root']
class Left(Root):
    def method(self): return ['left'] + super().method()
class Right(Root):
    def method(self): return ['right'] + super().method()
class Diamond(Left, Right): pass
obj = Diamond()
assert obj.method() == ['left', 'right', 'root']
assert super(Left, obj).method() == ['right', 'root']
proxy = super(Left, obj)
assert proxy.__thisclass__ is Left
assert proxy.__self__ is obj
assert proxy.__self_class__ is Diamond
class Captured(Left):
    def method(self):
        def capture(): return self
        assert capture() is self
        return super().method()
assert Captured().method() == ['left', 'root']
class Created:
    def __new__(cls, value):
        result = super().__new__(cls)
        result.value = value
        return result
    def __init__(self, value): self.initialized = value
created = Created(12)
assert created.value == 12 and created.initialized == 12
assert created.__new__ is Created.__new__
class Foreign:
    def __new__(cls): return 42
    def __init__(self): assert False
assert Foreign() == 42
static_calls = []
class StaticInit:
    @staticmethod
    def __init__(value): static_calls.append(value)
StaticInit(9)
assert static_calls == [9]
class BadInit:
    def __init__(self): return 42
try: BadInit()
except TypeError: pass
else: assert False
class BrokenDescriptor:
    def __set_name__(self, owner, name): raise ValueError('original')
try:
    class Bad:
        broken = BrokenDescriptor()
except ValueError as error:
    assert error.args == ('original',)
    assert 'Bad' in error.__notes__[0]
    assert 'broken' in error.__notes__[0]
else: assert False
for call in (lambda: super(1, obj), lambda: super(obj, obj), lambda: super(Left, 1), lambda: super(), lambda: type(name='X', bases=(), dict={})):
    try: call()
    except (TypeError, RuntimeError): pass
    else: assert False
class DynamicBase:
    def __init_subclass__(cls, flag=None): cls.flag = flag
kw = {'flag': 123}
meta, ns, remaining = types.prepare_class('Dynamic', (DynamicBase,), kw)
assert meta is type and remaining == {'flag': 123} and kw == {'flag': 123}
created = types.new_class('Dynamic', (DynamicBase,), kw, lambda ns: ns.__setitem__('value', 7))
assert created.flag == 123 and created.value == 7
assert kw == {'flag': 123}
doc = 'finished'
