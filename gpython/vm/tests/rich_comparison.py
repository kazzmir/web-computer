doc = 'comparison dispatch, arbitrary results, callable instances and recursion'
import operator
import sys
from collections import UserList
calls = []
class Left:
    def __lt__(self, other):
        calls.append('left')
        return NotImplemented
class Right(Left):
    def __gt__(self, other):
        calls.append('right')
        return 'rich result'
assert operator.lt(Left(), Right()) == 'rich result'
assert calls == ['right']
class EqBase:
    def __eq__(self, other): return 'base'
class EqChild(EqBase):
    def __eq__(self, other): return 'child'
assert operator.eq(EqBase(), EqChild()) == 'child'
assert operator.eq(EqChild(), EqBase()) == 'child'
class Equal:
    def __eq__(self, other): return True
assert Equal() == Equal()
assert not (Equal() != Equal())
class Declines:
    def __eq__(self, other): return NotImplemented
    def __ne__(self, other): return NotImplemented
one, two = Declines(), Declines()
assert one == one and not (one == two)
assert not (one != one) and one != two
class Reflected:
    def __eq__(self, other): return 'reflected'
assert operator.eq(one, Reflected()) == 'reflected'
class Truth:
    def __bool__(self): raise RuntimeError('truth failure')
class Rich:
    def __eq__(self, other): return Truth()
    def __lt__(self, other): return Truth()
assert isinstance(operator.eq(Rich(), Rich()), Truth)
for call in (lambda: bool(operator.eq(Rich(), Rich())), lambda: [Rich()] == [Rich()]):
    try: call()
    except RuntimeError: pass
    else: assert False
class Zero:
    def __eq__(self, other): return 0
assert [Zero()] != [Zero()]
assert bool() is False
assert bool([]) is False and bool([1]) is True
assert bool(None) is False and bool(0) is False
assert bool(1) is True and bool('yes') is True
assert issubclass(bool, int)
assert False < True and True > False
assert True == 1 and False == 0 and True != 2
assert False <= False and True >= True
class BadBool:
    def __bool__(self): return 1
class BadLen:
    def __len__(self): return -1
for value, error in ((BadBool(), TypeError), (BadLen(), ValueError)):
    try: bool(value)
    except error: pass
    else: assert False
try:
    class SubBool(bool): pass
except TypeError: pass
else: assert False
assert [1, 2] < [1, 3]
assert [1] < [1, 0]
assert [2] > [1, 99]
assert [1, 2] <= [1, 2]
assert [1, 2] >= [1, 2]
class Callable:
    def __call__(self, a, *, b=1): return a + b
assert Callable()(2, b=3) == 5
original = sys.getrecursionlimit()
sys.setrecursionlimit(60)
assert sys.getrecursionlimit() == 60
def recurse(): return recurse()
try: recurse()
except RecursionError: pass
else: assert False
# The accounting must unwind even when the interpreter raises while entering
# another frame, and a failed limit change must not corrupt the old limit.
assert Callable()(1) == 2
for value, error in ((0, ValueError), (-1, ValueError), (1, RecursionError), (1.5, TypeError), (1 << 100, OverflowError)):
    try: sys.setrecursionlimit(value)
    except error: pass
    else: assert False
assert sys.getrecursionlimit() == 60
class Index:
    def __index__(self): return 70
sys.setrecursionlimit(Index())
assert sys.getrecursionlimit() == 70
sys.setrecursionlimit(original)
a = UserList([1, [2]])
b = a.copy()
assert a == b and a is not b
assert a[1] is b[1]
assert a[:] == a and a[:] is not a
for position, expected in ((-99, [9, 1, 2]), (-1, [1, 9, 2]), (1, [1, 9, 2]), (99, [1, 2, 9])):
    value = [1, 2]
    assert value.insert(position, 9) is None
    assert value == expected
import random
value = list(range(30))
random.seed(123)
random.shuffle(value)
assert sorted(value) == list(range(30))
again = list(range(30))
random.seed(123)
random.shuffle(again)
assert again == value
doc = 'finished'
