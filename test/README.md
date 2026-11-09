# CPython regression tests

These test files are copied unchanged from
`/home/ubuntu/tmp/Python-3.15.2/Lib/test`:

| File | Cases | Coverage |
| --- | ---: | --- |
| `test_unary.py` | 6 | Unary arithmetic, complex numbers, big integers, precedence, invalid operands |
| `test_longexp.py` | 1 | Compile and evaluate a list expression with 65,580 elements |
| `test_keyword.py` | 11 | Reserved/soft keywords, name rejection, sorted lists, independent membership snapshots |
| `test_generator_stop.py` | 2 | PEP 479 wrapping and exception cause/context/suppression |
| `test_int_literal.py` | 6 | Hexadecimal, octal, binary, signed/unsigned boundaries, arbitrary-precision constants |
| `test_colorsys.py` | 8 | RGB/HSV/HLS/YIQ values, round trips, clamping, near-white precision |
| `test_keywordonlyarg.py` | 11 | Keyword-only binding/defaults, syntax validation, large signatures, mangling, evaluation order |
| `test_pow.py` | 7 | Integer/float powers, modular inverses, negative moduli, reflected power, huge random exponents |
| `test_typechecks.py` | 6 | Metaclass `__instancecheck__`/`__subclasscheck__`, virtual types, inheritance, tuple class arguments |
| `test_unicode_identifiers.py` | 3 | NFKC identifier normalization, non-BMP names, variation selectors, invalid-character diagnostics |
| `test_utf8source.py` | 3 | UTF-8 source/text, malformed source, byte-source Latin-1 encoding declarations |
| `test_html.py` | 2 | HTML escaping, HTML5 named/numeric references, invalid code points, long/repeated references |
| `test_secrets.py` | 11 | Digest comparison, secure random integers/choices, byte/hex/URL-safe tokens |
| `test_flufl.py` | 4 | Explicit compiler flags, future directives, comparison syntax, detailed diagnostics |

The upstream license is retained in `LICENSE.cpython`. The runtime also embeds
unchanged copies of `Lib/keyword.py` and `Lib/colorsys.py` from the same
distribution. Async/await
are reserved names; async statements themselves are not implemented yet.
`__future__` advertises `generator_stop` and `barry_as_FLUFL`. PEP 479 behavior
is mandatory even without its import. The latter is an opt-in comparison mode;
ordinary code continues to use `!=` and rejects `<>`.

Run the tests against this repository's Python runtime (not the host Python):

```sh
go test ./test -v
go run ./gpython test/test_unary.py
go run ./gpython test/test_longexp.py
```

`runtime_test.go` checks exact case counts to guard against empty discovery.
Each file runs in a fresh interpreter context. No cases are skipped or weakened.
The copied `test/__init__.py`, `tokenizedata/__init__.py`, and malformed-source
fixtures support the upstream `test.*` imports. Syntax-error fixtures are data,
not additional runnable test suites. Basic source/text codecs currently cover
UTF-8, Latin-1, and ASCII; unsupported encodings fail explicitly.

The `secrets` subset uses Go's `crypto/rand` and `crypto/hmac`, not the
simulation-oriented `random` module. It supplies the tested public functions;
`SystemRandom` and general buffer-protocol digest arguments are not provided.
Digest comparison supports ASCII `str` and `bytes`.

The runtime currently supplies a **limited, fail-fast `unittest` subset**:
`TestCase`, `setUp`/`tearDown`, LIFO `addCleanup`/`doCleanups`, `assertTrue`,
`assertFalse`, `assertEqual`, `assertAlmostEqual`, `assertIs`, `assertIn`,
`assertListEqual`, `assertIsInstance`, callable/context-manager
`assertRaises`/`assertRaisesRegex`, `assertRegex`, fail-fast `subTest` contexts, and module-level
`main()` discovery (including inherited test methods). Assertion failures and
unexpected exceptions propagate to the runner. This is not full CPython
`unittest`: it has no result aggregation, CLI options, skips, or test suites;
`main()` returns the number of cases on success instead of raising SystemExit.
Regex assertions currently accept Go/RE2 syntax, not all Python `re` patterns;
unsupported patterns fail explicitly rather than being silently ignored.
