// Package secrets provides cryptographic choices and tokens, without using
// the deterministic math/rand generator used by the random compatibility module.
package secrets

import (
	"crypto/hmac"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"math/big"

	"github.com/kazzmir/web-computer/gpython/py"
)

func nonnegativeSize(value py.Object) (int, error) {
	n, err := py.Index(value)
	if err != nil {
		return 0, err
	}
	if n < 0 {
		return 0, py.ExceptionNewf(py.ValueError, "number must be non-negative")
	}
	size := int(n)
	if py.Int(size) != n {
		return 0, py.ExceptionNewf(py.OverflowError, "number is too large")
	}
	return size, nil
}

func randomBytes(n int) (py.Bytes, error) {
	data := make(py.Bytes, n)
	if _, err := rand.Read(data); err != nil {
		return nil, py.ExceptionNewf(py.OSError, "secure randomness: %v", err)
	}
	return data, nil
}

func init() {
	py.RegisterModule(&py.ModuleImpl{
		Info:    py.ModuleInfo{Name: "secrets", FileDesc: "secrets.py", Doc: "Cryptographic randomness, token generation, and digest comparison"},
		Globals: py.StringDict{"DEFAULT_ENTROPY": py.Int(32)},
		Methods: []*py.Method{
			py.MustNewMethod("_randbelow", func(self py.Object, args py.Tuple) (py.Object, error) {
				var value py.Object
				if err := py.UnpackTuple(args, nil, "randbelow", 1, 1, &value); err != nil {
					return nil, err
				}
				bound, ok := py.ConvertToBigInt(value)
				if !ok {
					index, err := py.Index(value)
					if err != nil {
						return nil, err
					}
					bound, _ = py.ConvertToBigInt(index)
				}
				if (*big.Int)(bound).Sign() <= 0 {
					return nil, py.ExceptionNewf(py.ValueError, "upper bound must be positive")
				}
				n, err := rand.Int(rand.Reader, (*big.Int)(bound))
				if err != nil {
					return nil, py.ExceptionNewf(py.OSError, "secure randomness: %v", err)
				}
				return (*py.BigInt)(n).MaybeInt(), nil
			}, 0, ""),
			py.MustNewMethod("_randbits", func(self py.Object, args py.Tuple) (py.Object, error) {
				var value py.Object
				if err := py.UnpackTuple(args, nil, "randbits", 1, 1, &value); err != nil {
					return nil, err
				}
				bits, err := nonnegativeSize(value)
				if err != nil {
					return nil, err
				}
				if bits == 0 {
					return py.Int(0), nil
				}
				size := (bits-1)/8 + 1
				data, err := randomBytes(size)
				if err != nil {
					return nil, err
				}
				if remainder := uint(bits % 8); remainder != 0 {
					data[0] &= byte((1 << remainder) - 1)
				}
				return (*py.BigInt)(new(big.Int).SetBytes(data)).MaybeInt(), nil
			}, 0, ""),
			py.MustNewMethod("_token_bytes", func(self py.Object, args py.Tuple) (py.Object, error) {
				var value py.Object
				if err := py.UnpackTuple(args, nil, "token_bytes", 1, 1, &value); err != nil {
					return nil, err
				}
				size, err := nonnegativeSize(value)
				if err != nil {
					return nil, err
				}
				return randomBytes(size)
			}, 0, ""),
			py.MustNewMethod("_token_string", func(self py.Object, args py.Tuple) (py.Object, error) {
				var value, urlsafe py.Object
				if err := py.UnpackTuple(args, nil, "_token_string", 2, 2, &value, &urlsafe); err != nil {
					return nil, err
				}
				data, ok := value.(py.Bytes)
				if !ok {
					return nil, py.ExceptionNewf(py.TypeError, "token data must be bytes")
				}
				if urlsafe == py.True {
					return py.String(base64.RawURLEncoding.EncodeToString(data)), nil
				}
				return py.String(hex.EncodeToString(data)), nil
			}, 0, ""),
			py.MustNewMethod("compare_digest", compareDigest, 0, "Compare ASCII text or byte digests without content-dependent short circuiting."),
		},
		CodeSrc: `
def randbelow(exclusive_upper_bound): return _randbelow(exclusive_upper_bound)
def randbits(k): return _randbits(k)
def choice(sequence):
    if not len(sequence): raise IndexError('cannot choose from an empty sequence')
    return sequence[_randbelow(len(sequence))]
def token_bytes(nbytes=None):
    return _token_bytes(DEFAULT_ENTROPY if nbytes is None else nbytes)
def token_hex(nbytes=None): return _token_string(token_bytes(nbytes), False)
def token_urlsafe(nbytes=None): return _token_string(token_bytes(nbytes), True)
`,
	})
}

func compareDigest(self py.Object, args py.Tuple) (py.Object, error) {
	var a, b py.Object
	if err := py.UnpackTuple(args, nil, "compare_digest", 2, 2, &a, &b); err != nil {
		return nil, err
	}
	var first, second []byte
	switch value := a.(type) {
	case py.String:
		other, ok := b.(py.String)
		if !ok {
			return nil, py.ExceptionNewf(py.TypeError, "digest arguments must have the same type")
		}
		first, second = []byte(value), []byte(other)
		for _, data := range [][]byte{first, second} {
			for _, c := range data {
				if c > 127 {
					return nil, py.ExceptionNewf(py.TypeError, "comparing non-ASCII strings is not supported")
				}
			}
		}
	case py.Bytes:
		other, ok := b.(py.Bytes)
		if !ok {
			return nil, py.ExceptionNewf(py.TypeError, "digest arguments must have the same type")
		}
		first, second = value, other
	default:
		return nil, py.ExceptionNewf(py.TypeError, "digests must be ASCII strings or bytes")
	}
	return py.NewBool(hmac.Equal(first, second)), nil
}
