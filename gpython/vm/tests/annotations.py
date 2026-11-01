doc = 'named variable annotations and runtime typing metadata'
from typing import List
trace = []
def value():
    trace.append('value')
    return 4
def annotation():
    trace.append('annotation')
    return int
x: annotation() = value()
y: annotation()
assert x == 4
assert 'y' not in globals()
assert __annotations__['x'] is int and __annotations__['y'] is int
assert trace == ['value', 'annotation', 'annotation']
class Sprite:
    pass
class Actor(Sprite):
    images: List[int] = []
    title: str
assert Actor.images == []
assert Actor.__annotations__['images'].__origin__ is list
assert Actor.__annotations__['images'].__args__ == (int,)
assert Actor.__annotations__['title'] is str
assert isinstance(Actor(), Sprite)
assert isinstance(Actor(), (str, Sprite))

def local_annotations():
    x: missing_type = 7
    y: missing_type
    return x
assert local_annotations() == 7
class Collection:
    def __len__(self): return 0
class ChildCollection(Collection):
    pass
collection = ChildCollection()
assert len(collection) == 0
assert not collection
collection.__len__ = lambda: 99
assert len(collection) == 0
assert not collection
assert True - False == 1
assert False - True == -1
assert True - True == 0
values = [1, 2, 1]
values.remove(1)
assert values == [2, 1]
try:
    values.remove(4)
except ValueError:
    pass
else:
    assert False
doc = 'finished'
