doc = 'f-string literals, expressions, and concatenation'
name = 'world'
assert f'hello {name}!' == 'hello world!'
assert F'{1 + 2 * 3}' == '7'
assert f'{{{name}}}' == '{world}'
assert f'{{}}' == '{}'
assert f'' == ''
assert f'no fields' == 'no fields'
assert f'{name}' '!' f' {2}' == 'world! 2'
assert 'hello ' f'{name}' == 'hello world'
assert rf'\n{name}' == '\\nworld'
assert Fr'{name}\n' == 'world\\n'
assert f'\n{name}\t' == '\nworld\t'
assert f'\u007b{name}\u007d' == '{world}'
assert f'\{name}' == '\\world'
assert f'''first
{name}
last''' == 'first\nworld\nlast'
assert f'{ {"key": 3}["key"] }' == '3'
assert f'{[x * 2 for x in range(3)][1]}' == '2'
assert f'{(lambda n: n + 1)(2)}' == '3'
assert f'{1 != 2} {1 == 2} {1 >= 2} {1 <= 2}' == 'True False False True'
assert f'{"a:b!c"}' == 'a:b!c'
assert f'{f"inner {name}"}' == 'inner world'

doc = 'format specs, conversions, and nested fields'
assert f'{1.234567:.4f}' == '1.2346'
assert f'{12:04d}' == '0012'
assert f'{-12:05d}' == '-0012'
assert f'{255:#06x}' == '0x00ff'
assert f'{1234567:,d}' == '1,234,567'
assert f'{0.125:.1%}' == '12.5%'
assert f'{12.5:+.2f}' == '+12.50'
assert f'{2.0:.0f}' == '2'
assert f'{2.0:#.0f}' == '2.'
assert f'{"cat":*^7}' == '**cat**'
assert f'{"elephant":.3s}' == 'ele'
assert f'{"é":>3}' == '  é'
assert f'{"é"!a}' == "'\\xe9'"
assert f'{"é"!a}' == ascii('é')
assert f'{"🐍"!a}' == "'\\U0001f40d'"
assert f'{"hello"!r:>9}' == "  'hello'"
width, precision = 8, 2
assert f'{3.14159:{width}.{precision}f}' == '    3.14'
assert f'{name=}' == "name='world'"
assert f'{1 + 2 = }' == '1 + 2 = 3'
assert f'{1.25=:.1f}' == '1.25=1.2'
assert format(1.234567, '.4f') == '1.2346'
assert format('a', '>3') == '  a'
assert format(10) == '10'
assert format(2**80, 'x') == '100000000000000000000'

doc = 'local scope, closures, comprehension scopes, and builtin shadowing'
def closure(x):
    return lambda: f'value={x}'
assert closure(42)() == 'value=42'
assert [f'{n}' for n in range(3)] == ['0', '1', '2']
def shadowed():
    str = repr = format = None
    local = 12
    return f'{local} {local!s} {local!r} {local:04d}'
assert shadowed() == '12 12 12 0012'

doc = 'special methods and evaluation order'
trace = []
class Value:
    def __str__(self):
        trace.append('str')
        return 'value'
    def __repr__(self): return 'repr-value'
    def __format__(self, spec):
        trace.append('format:' + spec)
        return 'custom:' + spec
v = Value()
assert f'{v}' == 'custom:'
assert f'{v:spec}' == 'custom:spec'
assert f'{v:{{}}}' == 'custom:{}'
assert f'{v!s} {v!r}' == 'value repr-value'

def next_value():
    trace.append('next')
    return 2
trace = []
assert f'{v!s}{next_value()}' == 'value2'
assert trace == ['str', 'next']
trace = []
assert f'{v}{next_value()}' == 'custom:2'
assert trace == ['format:', 'next']
class BadFormat:
    def __format__(self, spec): return 3
try:
    f'{BadFormat()}'
except TypeError:
    pass
else:
    assert False
for spec in ('bad', '.f', '.2d', '+s', '>3s'):
    try:
        format(3, spec)
    except ValueError:
        pass
    else:
        assert False

try:
    f'{unknown_fstring_name}'
except NameError:
    pass
else:
    assert False
try:
    f'{1 / 0}'
except ZeroDivisionError:
    pass
else:
    assert False

doc = 'finished'
