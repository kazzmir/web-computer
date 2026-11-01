doc = 'PEP 479 is mandatory, preserves exception chaining, and closes failed generators'

def normal():
    if False: yield
try:
    next(normal())
except StopIteration:
    pass
else:
    assert False

assert type(StopIteration) is type
class CustomError(ValueError):
    def __init__(self, value): self.value = value
custom = CustomError(42)
assert custom.value == 42
assert custom.args == (42,)
assert isinstance(custom, ValueError)
class CustomStop(StopIteration): pass
cause = CustomStop('finished')
def failing():
    yield 1
    raise cause
it = failing()
assert next(it) == 1
try:
    next(it)
except RuntimeError as error:
    assert error.__cause__ is cause
    assert error.__context__ is cause
    assert error.__suppress_context__ is True
    assert str(error) == 'generator raised StopIteration'
else:
    assert False
try:
    next(it)
except StopIteration:
    pass
else:
    assert False

def caught():
    try:
        raise StopIteration
    except StopIteration:
        yield 'recovered'
assert list(caught()) == ['recovered']

def other_error():
    yield 1
    raise ValueError('other')
it = other_error()
assert next(it) == 1
try:
    next(it)
except ValueError:
    pass
else:
    assert False
try:
    next(it)
except StopIteration:
    pass
else:
    assert False

assert str(StopIteration()) == ''
assert str(StopIteration(3)) == '3'
assert str(StopIteration(1, 2)) == '(1, 2)'
assert StopIteration().__cause__ is None
assert StopIteration().__context__ is None
assert StopIteration().__suppress_context__ is False
import __future__
generator_stop = __future__.generator_stop
assert generator_stop.getMandatoryRelease() == (3, 7, 0, 'alpha', 0)
compile('answer = 42', '<generator_stop flag>', 'exec', generator_stop.compiler_flag)
doc = 'finished'
