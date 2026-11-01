doc = 'keyword-only defaults remain aligned for functions and lambdas'
def f(*, first=1, required, last=3):
    return first, required, last
assert f(required=2) == (1, 2, 3)
assert f.__kwdefaults__ == {'first': 1, 'last': 3}
assert f.__code__.co_argcount == 0
assert f.__code__.co_kwonlyargcount == 3
fn = lambda *, required1, default=2, required2: (required1, default, required2)
assert fn(required1=1, required2=3) == (1, 2, 3)
assert fn.__kwdefaults__ == {'default': 2}
for text in ('def f(*): pass', 'def f(*, **kw): pass', 'lambda *: None', 'lambda *, **kw: None'):
    try:
        compile(text, '<bare star>', 'exec')
    except SyntaxError:
        pass
    else:
        assert False

def named(a, *, b): pass
named.__qualname__ = 'custom.qualified'
for call in (lambda: named(1, 2), lambda: named(), lambda: named(1), lambda: named(1, bad=2), lambda: named(1, a=2)):
    try:
        call()
    except TypeError as error:
        assert str(error).startswith('custom.qualified()')
    else:
        assert False

assert ','.join(str(i) for i in range(3)) == '0,1,2'
assert '-'.join(('ä', '蟒')) == 'ä-蟒'
assert '.'.join([]) == ''
try:
    ','.join(['a', 1])
except TypeError:
    pass
else:
    assert False
try:
    ','.join(1)
except TypeError:
    pass
else:
    assert False
doc = 'finished'
