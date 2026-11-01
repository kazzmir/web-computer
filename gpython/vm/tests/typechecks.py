doc = 'metaclass hooks, preparation, introspection, and stable class bases'
class Base: pass
class Child(Base): pass
# Calls reuse the VM stack; class bases must not alias its old argument slice.
for i in range(20): str(i)
assert Child.mro() == [Child, Base, object]
assert type.mro(Child) == [Child, Base, object]
assert issubclass(Child, (str, Base))
assert not issubclass(Base, Child)
assert not issubclass(Child, ())
for call in (lambda: issubclass(1, object), lambda: issubclass(Child, 1), lambda: issubclass(Child, (str, 1)), lambda: type.mro(1)):
    try: call()
    except TypeError: pass
    else: assert False

class Meta(type):
    def __instancecheck__(cls, value): return 1 if type(value) is str else 0
    def __subclasscheck__(cls, value): return 1 if value is int else 0
    @property
    def priority(cls): return 'metaclass'
class Virtual(metaclass=Meta): priority = 'class'
assert type(Virtual) is Meta
assert isinstance('text', Virtual)
assert not isinstance(3, Virtual)
assert isinstance(Virtual(), Virtual)  # exact instances bypass the hook
assert issubclass(int, Virtual)
assert not issubclass(str, Virtual)
assert Virtual.priority == 'metaclass'
assert Virtual.__dict__['priority'] == 'class'
assert Virtual().__class__ is Virtual
assert (1).__class__ is int
assert Base().__dict__ == {}

class Reject(type):
    def __instancecheck__(cls, value): raise ValueError('instance hook')
    def __subclasscheck__(cls, value): raise ValueError('subclass hook')
class Rejected(metaclass=Reject): pass
for call in (lambda: isinstance(1, Rejected), lambda: issubclass(int, Rejected)):
    try: call()
    except ValueError: pass
    else: assert False

class PreparedMeta(type):
    @classmethod
    def __prepare__(cls, name, bases): return {'seed': 42}
    def mro(cls): return type.mro(cls)
class Prepared(metaclass=PreparedMeta): result = seed
assert Prepared.result == 42
assert Prepared.mro() == [Prepared, object]
def factory(name, bases, namespace): return namespace['result']
class ByFactory(metaclass=factory): result = 17
assert ByFactory == 17
doc = 'finished'
