package io_test

import (
	"github.com/kazzmir/web-computer/gpython/py"
	_ "github.com/kazzmir/web-computer/gpython/stdlib"
	"testing"
)

func TestTextStreamsAndPrintErrors(t *testing.T) {
	opts := py.DefaultContextOpts()
	opts.SysPaths = append(append([]string(nil), opts.SysPaths...), "../../..")
	ctx := py.NewContext(opts)
	defer ctx.Close()
	code, err := py.Compile(`
from io import StringIO
from test import support
import sys
s = StringIO('ab😀d')
assert s.read(3) == 'ab😀'
assert s.tell() == 3
assert s.write('Z') == 1
assert s.getvalue() == 'ab😀Z'
assert s.seek(0) == 0
assert s.read() == 'ab😀Z'
assert s.seek(7) == 7
assert s.write('') == 0
assert s.getvalue() == 'ab😀Z'
assert s.write('x') == 1
assert s.getvalue() == 'ab😀Z\0\0\0x'
assert s.seek(0, 2) == 8
assert s.seek(0, 1) == 8
assert s.read() == ''
s.close()
s.close()
assert s.closed
for call in (s.getvalue, s.flush, s.tell, lambda: s.read(), lambda: s.write(''), lambda: s.seek(0)):
    try: call()
    except ValueError: pass
    else: assert False
try: s.closed = False
except AttributeError: pass
else: assert False
with StringIO() as stream:
    print('é', '😀', file=stream, sep=None, end=None)
    assert stream.getvalue() == 'é 😀\n'
assert stream.closed
original = sys.stdout
try:
    with support.captured_stdout() as output:
        print('captured')
        raise ValueError('restore')
except ValueError: pass
assert sys.stdout is original
assert output.getvalue() == 'captured\n'
with support.captured_stdout() as output:
    assert print('default', file=None, sep=None, end=None) is None
assert output.getvalue() == 'default\n'
class FlushResult:
    def write(self, text): pass
    def flush(self): return 123
assert print(file=FlushResult(), flush=True) is None
class MissingFlush:
    def write(self, text): pass
try: print('x', file=MissingFlush(), flush=True)
except AttributeError: pass
else: assert False
class BadBool:
    def __bool__(self): raise RuntimeError('truth failure')
with support.captured_stdout() as output:
    try: print('not written', flush=BadBool())
    except RuntimeError: pass
    else: assert False
assert output.getvalue() == ''
class BadString:
    def __str__(self): return 1
with support.captured_stdout():
    try: print(BadString())
    except TypeError: pass
    else: assert False
with support.swap_attr(sys, 'stdout', None):
    assert print('ignored', sep=123) is None
    try: print('ignored', flush=BadBool())
    except RuntimeError: pass
    else: assert False
class Holder: pass
holder = Holder()
with support.swap_attr(holder, 'temporary', 42):
    assert holder.temporary == 42
assert not hasattr(holder, 'temporary')
support.gc_collect()
`, "StringIO regression", py.ExecMode, 0, false)
	if err == nil {
		_, err = py.RunCode(ctx, code, "StringIO regression", nil)
	}
	if err != nil {
		py.TracebackDump(err)
		t.Fatal(err)
	}
}
