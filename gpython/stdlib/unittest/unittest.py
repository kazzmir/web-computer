# Intentionally small compatibility subset, not the full CPython unittest.
# Failures propagate to the embedding runner; no test is silently skipped.
class _AssertRaisesContext:
    def __init__(self, expected):
        self.expected = expected
        self.exception = None
    def __enter__(self): return self
    def __exit__(self, kind, value, traceback):
        if kind is None:
            raise AssertionError('expected exception was not raised')
        if not isinstance(value, self.expected): return False
        self.exception = value
        return True

class TestCase:
    def __init__(self, methodName='runTest'): self._testMethodName = methodName
    def setUp(self): pass
    def tearDown(self): pass
    def run(self):
        self.setUp()
        try:
            getattr(self, self._testMethodName)()
        finally:
            self.tearDown()
    def fail(self, msg=None):
        raise AssertionError(msg if msg is not None else 'test failed')
    def assertTrue(self, expr, msg=None):
        if not expr: self.fail(msg if msg is not None else 'expected a true value')
    def assertFalse(self, expr, msg=None):
        if expr: self.fail(msg if msg is not None else 'expected a false value')
    def assertEqual(self, first, second, msg=None):
        if first != second:
            self.fail(msg if msg is not None else repr(first) + ' != ' + repr(second))
    def assertIsInstance(self, obj, cls, msg=None):
        if not isinstance(obj, cls): self.fail(msg if msg is not None else 'unexpected instance type')
    def assertRaises(self, expected, callable=None, *args, **kwargs):
        context = _AssertRaisesContext(expected)
        if callable is None: return context
        with context:
            callable(*args, **kwargs)

_tests_run = 0

def main(module='__main__'):
    global _tests_run
    _tests_run = 0
    tests = _discover(module)
    if not tests: raise AssertionError('no tests discovered')
    for case, method in tests:
        _tests_run += 1
        case(method).run()
    print('Ran %d tests (OK)' % _tests_run)
    return _tests_run
