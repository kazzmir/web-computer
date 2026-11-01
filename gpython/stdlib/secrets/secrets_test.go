package secrets_test

import (
	"testing"

	"github.com/kazzmir/web-computer/gpython/py"
	_ "github.com/kazzmir/web-computer/gpython/stdlib"
)

func TestSecureRandomBoundsAndValidation(t *testing.T) {
	ctx := py.NewContext(py.DefaultContextOpts())
	defer ctx.Close()
	code, err := py.Compile(`
import secrets
for bits in (0, 1, 7, 8, 9, 255, 1000):
    for i in range(10):
        assert 0 <= secrets.randbits(bits) < (1 << bits)
assert secrets.randbelow(1) == 0
for i in range(10):
    bound = (1 << 1000) + 123
    assert 0 <= secrets.randbelow(bound) < bound
assert secrets.choice(('only',)) == 'only'
assert secrets.token_bytes(0) == b''
assert len(secrets.token_bytes(0)) == 0 and not secrets.token_bytes(0)
assert secrets.token_hex(0) == ''
assert secrets.token_urlsafe(0) == ''
for n in range(1, 20):
    assert len(secrets.token_urlsafe(n)) == (8 * n + 5) // 6
assert secrets.compare_digest('', '') is True
assert secrets.compare_digest(b'\x00\xff', b'\x00\xff') is True
assert secrets.compare_digest(b'\x00\xff', b'\x00\xfe') is False
for call in (lambda: secrets.randbits(-1), lambda: secrets.randbelow(0), lambda: secrets.token_bytes(-1)):
    try: call()
    except ValueError: pass
    else: assert False
for call in (lambda: secrets.randbits(1.5), lambda: secrets.randbelow(2.5), lambda: secrets.token_hex(1.5), lambda: secrets.compare_digest('é', 'é'), lambda: secrets.compare_digest('x', b'x'), lambda: secrets.compare_digest(1, 1)):
    try: call()
    except TypeError: pass
    else: assert False
try: secrets.choice([])
except IndexError: pass
else: assert False
`, "secrets regression", py.ExecMode, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := py.RunCode(ctx, code, "secrets regression", nil); err != nil {
		py.TracebackDump(err)
		t.Fatal(err)
	}
}
