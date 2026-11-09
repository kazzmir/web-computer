class UserList:
    def __init__(self, initlist=None):
        self.data = [] if initlist is None else list(initlist)
    def __len__(self): return len(self.data)
    def __getitem__(self, key):
        value = self.data[key]
        return self.__class__(value) if isinstance(key, slice) else value
    def __setitem__(self, key, value): self.data[key] = value
    def __delitem__(self, key): del self.data[key]
    def __iter__(self): return iter(self.data)
    def __contains__(self, value): return value in self.data
    def __repr__(self): return repr(self.data)
    def _cast(self, other): return other.data if isinstance(other, UserList) else other
    def __eq__(self, other): return self.data == self._cast(other)
    def __ne__(self, other): return self.data != self._cast(other)
    def __lt__(self, other): return self.data < self._cast(other)
    def __le__(self, other): return self.data <= self._cast(other)
    def __gt__(self, other): return self.data > self._cast(other)
    def __ge__(self, other): return self.data >= self._cast(other)
    def append(self, value): self.data.append(value)
    def insert(self, index, value): self.data.insert(index, value)
    def extend(self, values): self.data.extend(values)
    def pop(self, index=-1): return self.data.pop(index)
    def clear(self): del self.data[:]
    def copy(self): return self.__class__(self.data)
