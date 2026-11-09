doc = 'runtime unions, generic aliases and abstract type-check recursion'
import sys
import types
import typing
class Base: pass
class Child(Base): pass
assert Child.__bases__ == (Base,)
assert Child.__base__ is Base
assert Child.__mro__ == (Child, Base, object)
assert (int | str | int).__args__ == (int, str)
assert (int | str) == (str | int)
assert (int | int) is int
assert (int | None).__args__ == (int, type(None))
assert repr(int | None) == 'int | None'
assert isinstance(int | str, types.UnionType)
assert (int | str).__origin__ is typing.Union
assert isinstance(1, int | str)
assert isinstance(None, int | None)
assert not isinstance(1.5, str | int)
assert issubclass(Child, str | Base)
assert not issubclass(int, str | Base)
assert list[int].__origin__ is list
assert list[int].__args__ == (int,)
assert list[int] == list[int]
assert list[int] != list[str]
assert isinstance(list[int], types.GenericAlias)
assert (list[int] | list[int]) == list[int]
assert typing.List[int].__origin__ is list
assert typing.Tuple[int, str].__args__ == (int, str)
assert isinstance([], typing.List | typing.Tuple)
assert issubclass(typing.List, typing.List | typing.Tuple)
assert isinstance(1, int | list[str])
assert issubclass(int, int | list[str])
for call in (lambda: isinstance([], list[int]), lambda: isinstance(1, list[int] | int), lambda: issubclass(int, list[str] | int), lambda: issubclass(1, int | str)):
    try: call()
    except TypeError: pass
    else: assert False
class Meta(type):
    def __or__(cls, other): return 'custom or'
class Custom(metaclass=Meta): pass
assert Custom | int == 'custom or'
class Spoof:
    @property
    def __class__(self): return Child
assert isinstance(Spoof(), Base)
class Broken:
    @property
    def __class__(self): raise RuntimeError('class getter')
try: isinstance(Broken(), int)
except RuntimeError as error: assert error.args == ('class getter',)
else: assert False
class Abstract:
    def __init__(self, bases): self.bases = bases
    @property
    def __bases__(self): return self.bases
root = Abstract(())
leaf = Abstract((root,))
class Instance:
    @property
    def __class__(self): return leaf
assert isinstance(Instance(), root)
assert issubclass(leaf, root)
assert not issubclass(root, leaf)
original = sys.getrecursionlimit()
sys.setrecursionlimit(80)
class Cycle:
    @property
    def __bases__(self): return (self,)
assert not isinstance(1, Cycle())
for call in (lambda: issubclass(Cycle(), int),):
    try: call()
    except RecursionError: pass
    else: assert False
nested = (str,)
for i in range(100): nested = (nested,)
for check in (lambda: issubclass(int, nested), lambda: isinstance(1, nested)):
    try: check()
    except RecursionError: pass
    else: assert False
sys.setrecursionlimit(original)
assert issubclass(Child, Base)
assert isinstance(Child(), Base)
class Subscript:
    def __class_getitem__(cls, argument): return (cls, argument)
assert Subscript[int] == (Subscript, int)
class MetaGet(type):
    def __getitem__(cls, argument): return 'metaclass first'
class WithMeta(Subscript, metaclass=MetaGet): pass
assert WithMeta[int] == 'metaclass first'
doc = 'finished'
