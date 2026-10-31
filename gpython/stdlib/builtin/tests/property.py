class Box:
    def __init__(self): self._value = 3
    def get(self): return self._value
    def set(self, value): self._value = value
    def delete(self): self._value = None
    value = property(get, set, delete)
    readonly = property(get)

b = Box()
assert b.value == 3
b.value = 10
assert b.value == 10
assert b.readonly == 10
assert isinstance(Box.value, property)
del b.value
assert b.value is None
try:
    b.readonly = 1
except AttributeError:
    pass
else:
    assert False

doc = 'finished'
