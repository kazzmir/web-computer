doc = 'method metadata, general-key caches, formatting and assignment expressions'
def fn(value):
    'function docs'
    return value
for wrap in (staticmethod, classmethod):
    method = wrap(fn)
    assert method.__func__ is fn
    assert method.__wrapped__ is fn
    assert method.__annotations__ is fn.__annotations__
    assert method.__module__ == fn.__module__
    assert method.__doc__ == 'function docs'
    assert method.__dict__ is method.__dict__
assert staticmethod(fn)(7) == 7
assert fn.__dict__ == {}
fn.__annotations__['value'] = int
assert staticmethod(fn).__annotations__['value'] is int
left, right = {}, {}
assert left is left and left is not right
one, two = tuple([1, 2]), tuple([1, 2])
assert one is one and one is not two
cache = {}
alias = cache
cache[(1, 2)] = 'tuple'
cache[3] = 'number'
assert alias[(True, 2.0)] == 'tuple'
assert alias[3] == 'number'
alias['name'] = 'text'
assert cache['name'] == 'text'
cache.update({4: 'four'}, extra=5)
assert cache[4] == 'four' and cache['extra'] == 5
copy = cache.copy()
assert copy == cache and copy is not cache
copy[3] = 'changed'
assert cache[3] == 'number'
created = dict()
created[1] = 2
assert created == {1: 2}
namespace = {}
exec('answer = 42', namespace)
assert namespace['answer'] == 42
namespace['answer'] = 43
assert eval('answer', namespace) == 43
namespace.update(extra=1)
assert eval('extra', namespace) == 1
assert __import__('math', {}, {}) is __import__('math')
def keywords(**kwargs): return kwargs
assert keywords(**{}) == {}
try: keywords(**{1: 2})
except TypeError: pass
else: assert False
assert '{} {name} {!r}'.format('plain', 'quoted', name=7) == "plain 7 'quoted'"
assert '{1}-{0}'.format('a', 'b') == 'b-a'
assert '{{{0:04d}}}'.format(12) == '{0012}'
assert '{value:{width}.{precision}f}'.format(value=1.25, width=6, precision=1) == '   1.2'
assert '{0[1]}'.format([10, 20]) == '20'
assert '{0.__name__}'.format(fn) == 'fn'
assert '{!a}'.format('é') == "'\\xe9'"
for format_string, error in (('{} {0}', ValueError), ('{', ValueError), ('}', ValueError), ('{!x}', ValueError), ('{1}', IndexError), ('{missing}', KeyError)):
    try: format_string.format('x')
    except error: pass
    else: assert False
class BadString:
    def __str__(self): return 1
try: '{!s}'.format(BadString())
except TypeError: pass
else: assert False
calls = []
def value():
    calls.append(1)
    return 12
assert (assigned := value()) == 12
assert assigned == 12 and calls == [1]
if test := 3:
    assert test == 3
else: assert False
n = 3
while n := n - 1:
    assert n > 0
assert n == 0
if False: assert False
elif other := 5:
    assert other == 5
else: assert False
def outer():
    state = 0
    def inner():
        nonlocal state
        return (state := state + 1)
    assert inner() == 1 and inner() == 2
    return state
assert outer() == 2
def identity(function): return function
@chosen := identity
def decorated(): return 'ok'
assert chosen is identity and decorated() == 'ok'
for source in ('x := 1', 'x = y := 1', '(obj.x := 1)', '(items[0] := 1)', '[(x := i) for i in range(2)]'):
    try: compile(source, '<assignment expression>', 'exec')
    except SyntaxError: pass
    else: assert False
class Problem(Exception):
    def __init__(self, value): Exception.__init__(self, 'problem', value)
problem = Problem(7)
assert problem.args == ('problem', 7)
assert str(problem) == "('problem', 7)"
doc = 'finished'
