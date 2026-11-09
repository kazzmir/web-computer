doc = 'in-place arithmetic, reflected division, matrix dispatch and value dictionaries'
class Number:
    def __iadd__(self, other): return NotImplemented
    def __add__(self, other): return ('fallback', other)
value = Number()
value += 7
assert value == ('fallback', 7)
class Reflected:
    def __rtruediv__(self, value): return ('reflected', value)
assert 1 / Reflected() == ('reflected', 1)
assert (2 ** 100) / Reflected() == ('reflected', 2 ** 100)
try: 1 / '2'
except TypeError: pass
else: assert False
class Matrix:
    def __matmul__(self, value): return 'left'
    def __rmatmul__(self, value): return 'base reflected'
    def __imatmul__(self, value): return NotImplemented
class Inherited(Matrix): pass
class Right(Matrix):
    def __rmatmul__(self, value): return 'derived reflected'
assert Matrix() @ Inherited() == 'left'
assert Matrix() @ Right() == 'derived reflected'
assert 2 @ Matrix() == 'base reflected'
m = Matrix()
m @= 1
assert m == 'left'
class Block(Matrix): __imatmul__ = None
try:
    m = Block()
    m @= 1
except TypeError: pass
else: assert False
class InvalidRight(Matrix): __rmatmul__ = []
try: Matrix() @ InvalidRight()
except TypeError: pass
else: assert False
for text in ('a,b += 1', '[a,b] *= 2', '(a,b) @= c'):
    try: compile(text, '<augassign>', 'exec')
    except SyntaxError: pass
    else: assert False
values = {1: 'first', True: 'second', 1.0: 'third', 'name': 2}
assert len(values) == 2
assert values[1] == 'third'
assert values[True] == 'third'
assert values.get(99, 'missing') == 'missing'
assert values.get(99) is None
assert 1 in values
assert 99 not in values
values[2] = 'two'
assert values[2] == 'two'
assert dict(values) == values
assert dict([[1, 'third'], ['name', 2], [2, 'two']]) == values
assert {i: i * 2 for i in range(4)} == {0: 0, 1: 2, 2: 4, 3: 6}
assert dict(a=1) == {'a': 1}
assert dict({'a': 1}) == {'a': 1}
pairs = {('a', 1): 'tuple', 2 ** 100: 'big', None: 'none'}
assert pairs[('a', True)] == 'tuple'
assert pairs[2 ** 100] == 'big'
assert pairs[None] == 'none'
assert len(list(pairs.keys())) == 3
assert len(list(pairs.values())) == 3
assert len(list(pairs.items())) == 3
single = {0: 1}
del single[0]
assert not single
assert single == {}
assert {} == single
try: compile('x = {[]: 1}', '<unhashable>', 'exec')
except SyntaxError: assert False
try: result = {[]: 1}
except TypeError: pass
else: assert False
for boundary in ('\n', '\r', '\r\n', '\v', '\f', '\x1c', '\x1d', '\x1e', '\x85', '\u2028', '\u2029'):
    text = 'a' + boundary + 'b' + boundary
    assert text.splitlines() == ['a', 'b']
    assert text.splitlines(keepends=True) == ['a' + boundary, 'b' + boundary]
assert ''.splitlines() == []
assert '\n'.splitlines() == ['']
assert '\r\n'.splitlines(True) == ['\r\n']
assert '\ud800\n'.splitlines() == ['\ud800']
doc = 'finished'
