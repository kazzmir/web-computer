doc = 'Unicode identifiers normalize without changing string values'
µ = 3
assert globals()['μ'] == 3
assert 'µ' != 'μ'
𝔘𝔫𝔦𝔠𝔬𝔡𝔢 = 4
assert Unicode == 4
assert 'Unicode' in dir()
ｉｆ = 5
assert globals()['if'] == 5

class Base:
    inherited = 1
class Child(Base): pass
value = Child()
value.own = 2
assert 'inherited' in dir(value) and 'own' in dir(value)
assert dir(value) == sorted(dir(value))
class Custom:
    def __dir__(self): return ('z', 'a')
assert dir(Custom()) == ['a', 'z']
for text, column in [('€ = 2', 1), ('é = 1; € = 2', 8)]:
    try:
        compile(text, 'badunicode.py', 'exec')
    except SyntaxError as error:
        assert error.offset == column
        assert error.lineno == 1
        assert str(error) == "invalid character '€' (U+20AC) (badunicode.py, line 1)"
    else:
        assert False
doc = 'finished'
