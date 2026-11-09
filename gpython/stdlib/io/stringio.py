class StringIO:
    def __init__(self, initial_value='', newline='\n'):
        if not isinstance(initial_value, str): raise TypeError('initial_value must be str')
        if newline != '\n': raise NotImplementedError('only newline="\\n" is supported')
        self._text = initial_value
        self._position = 0
        self._closed = False
    @property
    def closed(self): return self._closed
    def _check(self):
        if self.closed: raise ValueError('I/O operation on closed file')
    def write(self, text):
        self._check()
        if not isinstance(text, str): raise TypeError('string argument expected')
        count = len(text)
        if count == 0: return 0
        if self._position > len(self._text): self._text += '\0' * (self._position - len(self._text))
        self._text = self._text[:self._position] + text + self._text[self._position + count:]
        self._position += count
        return count
    def read(self, size=-1):
        self._check()
        if not isinstance(size, int): raise TypeError('integer argument expected')
        stop = len(self._text) if size < 0 else min(len(self._text), self._position + size)
        result = self._text[self._position:stop]
        self._position += len(result)
        return result
    def getvalue(self):
        self._check()
        return self._text
    def tell(self):
        self._check()
        return self._position
    def seek(self, offset, whence=0):
        self._check()
        if not isinstance(offset, int) or not isinstance(whence, int): raise TypeError('integer argument expected')
        if whence == 0:
            if offset < 0: raise ValueError('negative seek position')
            self._position = offset
        elif whence == 1 or whence == 2:
            if offset != 0: raise OSError('nonzero relative seeks are not supported')
            if whence == 2: self._position = len(self._text)
        else: raise ValueError('invalid whence')
        return self._position
    def flush(self): self._check()
    def close(self): self._closed = True
    def writable(self):
        self._check()
        return True
    def readable(self):
        self._check()
        return True
    def seekable(self):
        self._check()
        return True
    def __enter__(self):
        self._check()
        return self
    def __exit__(self, typ, value, traceback): self.close()
