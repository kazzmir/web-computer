package unittest_test

import (
	"testing"

	"github.com/kazzmir/web-computer/gpython/py"
	_ "github.com/kazzmir/web-computer/gpython/stdlib"
)

func TestAssertionsAndDiscovery(t *testing.T) {
	ctx := py.NewContext(py.DefaultContextOpts())
	defer ctx.Close()
	src := `
import unittest
case = unittest.TestCase()
case.assertEqual(2, 2.0)
case.assertTrue([1])
case.assertFalse([])
case.assertIsInstance(ValueError(), Exception)
def raise_type(): raise TypeError('expected')
case.assertRaises(TypeError, raise_type)
with case.assertRaises(TypeError) as caught:
    raise_type()
assert isinstance(caught.exception, TypeError)
with case.assertRaises(AssertionError): case.assertTrue(False)
with case.assertRaises(AssertionError): case.assertFalse(True)
with case.assertRaises(AssertionError): case.assertEqual(1, 2)
with case.assertRaises(AssertionError): case.assertIsInstance(1, str)
with case.assertRaises(AssertionError): case.assertRaises(TypeError, lambda: None)
with case.assertRaises(AssertionError):
    with case.assertRaises(TypeError): pass
# A mismatched exception must propagate, not count as a successful assertion.
with case.assertRaises(TypeError): case.assertRaises(ValueError, raise_type)
with case.assertRaises(TypeError):
    with case.assertRaises(ValueError): raise_type()

# Invalid isinstance class arguments must still be rejected, including tuples.
for invalid in (1, case, (str, 1)):
    with case.assertRaises(TypeError): isinstance(1, invalid)
assert isinstance(1, (int, 1))
assert isinstance(TypeError(), (ValueError, Exception))

cleanup_log = []
cleanup = unittest.TestCase()
cleanup.addCleanup(cleanup_log.append, 'first')
cleanup.addCleanup(raise_type)
cleanup.addCleanup(cleanup_log.append, 'last')
with case.assertRaises(TypeError): cleanup.doCleanups()
assert cleanup_log == ['last', 'first']
assert cleanup.doCleanups() is True
class SetupFailure(unittest.TestCase):
    def setUp(self):
        self.addCleanup(cleanup_log.append, 'setup cleanup')
        raise_type()
    def runTest(self): case.fail('body must not run after failed setup')
    def tearDown(self): case.fail('teardown must not run after failed setup')
with case.assertRaises(TypeError): SetupFailure().run()
assert cleanup_log[-1] == 'setup cleanup'
case.assertIn(1, [1, 2])
case.assertListEqual([1, 2], [1, 2])
with case.assertRaises(AssertionError): case.assertIn(3, [1, 2])
with case.assertRaises(AssertionError): case.assertListEqual([1], [2])
with case.assertRaises(AssertionError): case.assertListEqual((1,), [1])

log = []
class Base(unittest.TestCase):
    tag = 'base'
    def setUp(self):
        self.fresh = []
        log.append(self.tag + ':setup')
    def tearDown(self): log.append(self.tag + ':teardown')
    def test_z(self):
        self.assertEqual(self.fresh, [])
        self.fresh.append(1)
        log.append(self.tag + ':z')
class Derived(Base):
    tag = 'derived'
    def test_a(self):
        self.assertEqual(self.fresh, [])
        self.fresh.append(2)
        log.append(self.tag + ':a')
assert unittest.main() == 3
assert log == ['base:setup', 'base:z', 'base:teardown', 'derived:setup', 'derived:a', 'derived:teardown', 'derived:setup', 'derived:z', 'derived:teardown']

class Broken(unittest.TestCase):
    def test_failure(self): self.fail('deliberate')
    def tearDown(self): log.append('cleanup')
with case.assertRaises(AssertionError): Broken('test_failure').run()
assert log[-1] == 'cleanup'
# A failing suite must not turn into a success report.
with case.assertRaises(AssertionError): unittest.main()
`
	code, err := py.Compile(src, "unittest regression", py.ExecMode, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := py.RunCode(ctx, code, "unittest regression", nil); err != nil {
		py.TracebackDump(err)
		t.Fatal(err)
	}
}
