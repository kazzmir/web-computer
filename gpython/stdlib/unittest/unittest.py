# Intentionally small compatibility subset, not the full CPython unittest.
# Failures propagate to the embedding runner; no test is silently skipped.
class _AssertRaisesContext:
    def __init__(self, expected, regex=None):
        self.expected = expected
        self.regex = regex
        self.exception = None
    def __enter__(self): return self
    def __exit__(self, kind, value, traceback):
        if kind is None:
            raise AssertionError('expected exception was not raised')
        if not isinstance(value, self.expected): return False
        self.exception = value
        if self.regex is not None and not _regex_matches(self.regex, str(value)):
            raise AssertionError('exception message did not match ' + repr(self.regex))
        return True

class _SubTestContext:
    def __enter__(self): return self
    def __exit__(self, kind, value, traceback): return False

class TestCase:
    def __init__(self, methodName='runTest'):
        self._testMethodName = methodName
        self._cleanups = []
    def setUp(self): pass
    def tearDown(self): pass
    def addCleanup(self, function, *args, **kwargs):
        self._cleanups.append((function, args, kwargs))
    def doCleanups(self):
        error = None
        while self._cleanups:
            function, args, kwargs = self._cleanups[-1]
            del self._cleanups[-1]
            try:
                function(*args, **kwargs)
            except BaseException as exc:
                if error is None: error = exc
        if error is not None: raise error
        return True
    def run(self):
        try:
            self.setUp()
            try:
                getattr(self, self._testMethodName)()
            finally:
                self.tearDown()
        finally:
            self.doCleanups()
    def fail(self, msg=None):
        raise AssertionError(msg if msg is not None else 'test failed')
    def assertTrue(self, expr, msg=None):
        if not expr: self.fail(msg if msg is not None else 'expected a true value')
    def assertFalse(self, expr, msg=None):
        if expr: self.fail(msg if msg is not None else 'expected a false value')
    def assertEqual(self, first, second, msg=None):
        if first != second:
            self.fail(msg if msg is not None else repr(first) + ' != ' + repr(second))
    def subTest(self, msg=None, **params):
        # This runner remains fail-fast: never swallow subtest failures.
        return _SubTestContext()
    def assertAlmostEqual(self, first, second, places=None, msg=None, delta=None):
        if first == second: return
        if places is not None and delta is not None:
            raise TypeError('specify delta or places, not both')
        difference = abs(first - second)
        if delta is not None:
            if difference <= delta: return
        else:
            if places is None: places = 7
            if round(difference, places) == 0: return
        self.fail(msg if msg is not None else repr(first) + ' != ' + repr(second) + ' within tolerance')
    def assertIs(self, first, second, msg=None):
        if first is not second: self.fail(msg if msg is not None else 'objects are not identical')
    def assertIn(self, member, container, msg=None):
        if member not in container:
            self.fail(msg if msg is not None else repr(member) + ' not found in ' + repr(container))
    def assertNotIn(self, member, container, msg=None):
        if member in container:
            self.fail(msg if msg is not None else repr(member) + ' unexpectedly found in ' + repr(container))
    def assertCountEqual(self, first, second, msg=None):
        remaining = list(second)
        for item in first:
            for i in range(len(remaining)):
                if item == remaining[i]:
                    del remaining[i]
                    break
            else:
                self.fail(msg if msg is not None else 'element counts differ')
        if remaining: self.fail(msg if msg is not None else 'element counts differ')
    def assertListEqual(self, first, second, msg=None):
        if not isinstance(first, list) or not isinstance(second, list):
            self.fail(msg if msg is not None else 'expected two lists')
        self.assertEqual(first, second, msg)
    def assertHasAttr(self, obj, name, msg=None):
        if not hasattr(obj, name):
            self.fail(msg if msg is not None else repr(obj) + ' has no attribute ' + repr(name))
    def assertIsInstance(self, obj, cls, msg=None):
        if not isinstance(obj, cls): self.fail(msg if msg is not None else 'unexpected instance type')
    def assertRaises(self, expected, callable=None, *args, **kwargs):
        context = _AssertRaisesContext(expected)
        if callable is None: return context
        with context:
            callable(*args, **kwargs)
    def assertRegex(self, text, regex, msg=None):
        if not _regex_matches(regex, text):
            self.fail(msg if msg is not None else repr(regex) + ' not found in ' + repr(text))
    def assertRaisesRegex(self, expected, regex, callable=None, *args, **kwargs):
        context = _AssertRaisesContext(expected, regex)
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
