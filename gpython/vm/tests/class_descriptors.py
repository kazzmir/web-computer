doc = 'class attribute descriptors and inherited class factories'
class Factory:
    @classmethod
    def make(cls, value):
        result = cls()
        result.value = value
        return result
    @staticmethod
    def add(a, b): return a + b

class Derived(Factory):
    pass

assert Factory.make(3).value == 3
assert Factory().make(4).value == 4
assert isinstance(Derived.make(5), Derived)
assert Factory.add(1, 2) == 3
assert Factory().add(2, 3) == 5
assert Derived.add(3, 4) == 7
assert (1, 2, 3) + (4,) == (1, 2, 3, 4)
assert (1,) + (2, 3, 4) == (1, 2, 3, 4)
assert () + (1, 2) == (1, 2)
assert (1, 2) + () == (1, 2)
doc = 'finished'
