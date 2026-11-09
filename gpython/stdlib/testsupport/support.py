import sys
from io import StringIO

class swap_attr:
    def __init__(self, obj, name, value):
        self.obj = obj
        self.name = name
        self.value = value
    def __enter__(self):
        try:
            self.original = getattr(self.obj, self.name)
            self.existed = True
        except AttributeError:
            self.existed = False
        setattr(self.obj, self.name, self.value)
        return self.value
    def __exit__(self, typ, value, traceback):
        if self.existed: setattr(self.obj, self.name, self.original)
        else: delattr(self.obj, self.name)

def captured_stdout():
    return swap_attr(sys, 'stdout', StringIO())

# This runtime has no tracing facility, so tests already execute without tracing.
def no_tracing(function): return function

class infinite_recursion:
    def __init__(self, max_depth=None):
        self.max_depth = 50 if max_depth is None else max_depth
    def __enter__(self):
        self.original = sys.getrecursionlimit()
        sys.setrecursionlimit(_recursion_depth() + self.max_depth)
        return self
    def __exit__(self, typ, value, traceback):
        sys.setrecursionlimit(self.original)
    def __call__(self, function):
        def run(*args, **kwargs):
            with infinite_recursion(self.max_depth):
                return function(*args, **kwargs)
        return run
