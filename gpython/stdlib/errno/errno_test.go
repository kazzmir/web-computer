package errno_test

import (
	"github.com/kazzmir/web-computer/gpython/py"
	_ "github.com/kazzmir/web-computer/gpython/stdlib"
	"testing"
)

func TestErrorcodeIsolationAndUppercase(t *testing.T) {
	for _, source := range []string{
		`import errno
assert errno.EACCES in errno.errorcode
for name in ('EDOM', 'ERANGE', 'ENOENT', 'EAGAIN', 'EWOULDBLOCK', 'ENOTSUP', 'EOPNOTSUPP'):
    code = getattr(errno, name)
    assert getattr(errno, errno.errorcode[code]) == code
assert len(errno.errorcode) > 50
errno.errorcode[errno.EACCES] = 'changed'
`,
		`import errno
assert errno.errorcode[errno.EACCES] == 'EACCES'
assert 'ABC_123'.isupper()
assert 'ÑÀΣ'.isupper()
assert '\u2160'.isupper()
assert '\u24b6'.isupper()
for text in ('', '123_!', 'abc', 'ABc', '\u01c5', 'A\u0345', '\ud800'):
    assert not text.isupper()
assert 'A\ud800'.isupper()
`,
	} {
		ctx := py.NewContext(py.DefaultContextOpts())
		code, err := py.Compile(source, "errno regression", py.ExecMode, 0, false)
		if err == nil {
			_, err = py.RunCode(ctx, code, "errno regression", nil)
		}
		ctx.Close()
		if err != nil {
			py.TracebackDump(err)
			t.Fatal(err)
		}
	}
}
