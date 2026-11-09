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
