# Small, explicit ndarray subset for GL buffers and transformation matrices.
# No host pointers, native extensions, broadcasting, or arbitrary NumPy APIs.
class _DType:
    def __init__(self, name, size, fields=None):
        self.name, self.itemsize, self.fields = name, size, fields
    def __getitem__(self, name):
        for field, dtype, width in self.fields:
            if field == name: return _DType(field, dtype.itemsize * width)
        raise KeyError(name)

float32 = _DType('float32', 4)
float64 = _DType('float64', 8)
uint32 = _DType('uint32', 4)

def _dtype(value):
    if isinstance(value, _DType): return value
    if isinstance(value, (tuple, list)):
        return _DType('structured', sum([dtype.itemsize * width for name, dtype, width in value]), value)
    raise TypeError('unsupported dtype')

def _flat(data):
    if isinstance(data, ndarray): return list(data._data)
    if isinstance(data, (tuple, list)):
        out = []
        for value in data: out.extend(_flat(value))
        return out
    return [data]

class ndarray:
    def __init__(self, data, shape, dtype):
        self._data, self.shape, self.dtype = data, tuple(shape), _dtype(dtype)
        self.nbytes = self.dtype.itemsize
        for n in self.shape: self.nbytes *= n
        self.strides = (self.dtype.itemsize,)
        if len(self.shape) == 2: self.strides = (self.dtype.itemsize * self.shape[1], self.dtype.itemsize)
    def __len__(self): return self.shape[0]
    def __getitem__(self, key):
        if key is Ellipsis: return self
        if isinstance(key, str): return self._data[key]
        if isinstance(key, tuple): return self._data[key[0] * self.shape[1] + key[1]]
        if len(self.shape) == 2:
            return self._data[key*self.shape[1]:(key+1)*self.shape[1]]
        return self._data[key]
    def __setitem__(self, key, value):
        if key is Ellipsis:
            data = _flat(value)
            if len(data) != len(self._data): raise ValueError('array shape mismatch')
            self._data[:] = data
        elif isinstance(key, str):
            width = None
            for field, dtype, count in self.dtype.fields:
                if field == key: width = count
            if width is None: raise KeyError(key)
            data = _flat(value)
            if len(data) != self.shape[0] * width: raise ValueError('field shape mismatch')
            self._data[key] = data
        elif isinstance(key, tuple): self._data[key[0] * self.shape[1] + key[1]] = value
        else: self._data[key] = value
    def _transpose(self):
        if len(self.shape) != 2: return self
        rows, cols = self.shape
        return ndarray([self[row, col] for col in range(cols) for row in range(rows)], (cols, rows), self.dtype)
    T = property(_transpose)
    def tolist(self):
        if len(self.shape) == 2: return [self[row] for row in range(self.shape[0])]
        return list(self._data)
    def tobytes(self): return _array_bytes(self)

def array(data, dtype=float64):
    if isinstance(data, ndarray): return ndarray(list(data._data), data.shape, dtype)
    shape = (len(data),)
    if len(data) and isinstance(data[0], (tuple, list)):
        shape = (len(data), len(data[0]))
        for row in data:
            if len(row) != shape[1]: raise ValueError('ragged arrays are unsupported')
    return ndarray(_flat(data), shape, dtype)

def zeros(shape, dtype=float64):
    dtype = _dtype(dtype)
    if isinstance(shape, int): shape = (shape,)
    count = 1
    for n in shape:
        if n < 0: raise ValueError('negative dimensions')
        count *= n
    if dtype.fields is not None:
        data = {name: [0] * (count*width) for name, field_type, width in dtype.fields}
    else: data = [0] * count
    return ndarray(data, shape, dtype)

def eye(n, dtype=float64):
    result = zeros((n, n), dtype)
    for i in range(n): result[i, i] = 1
    return result

def dot(a, b):
    if len(a.shape) != 2 or len(b.shape) != 2 or a.shape[1] != b.shape[0]: raise ValueError('dot requires compatible matrices')
    result = zeros((a.shape[0], b.shape[1]), a.dtype)
    for row in range(a.shape[0]):
        for col in range(b.shape[1]):
            result[row, col] = sum([a[row, k] * b[k, col] for k in range(a.shape[1])])
    return result
