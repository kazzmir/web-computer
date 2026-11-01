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

The upstream license is retained in `LICENSE.cpython`. The runtime also embeds
unchanged copies of `Lib/keyword.py` and `Lib/colorsys.py` from the same
distribution. Async/await
are reserved names; async statements themselves are not implemented yet.
`__future__` currently advertises only `generator_stop`; PEP 479 behavior is
mandatory, including without that future import.

Run the tests against this repository's Python runtime (not the host Python):

```sh
go test ./test -v
go run ./gpython test/test_unary.py
go run ./gpython test/test_longexp.py
```

`runtime_test.go` checks exact case counts to guard against empty discovery.
Each file runs in a fresh interpreter context. No cases are skipped or weakened.

The runtime currently supplies a **limited, fail-fast `unittest` subset**:
`TestCase`, `setUp`/`tearDown`, LIFO `addCleanup`/`doCleanups`, `assertTrue`,
`assertFalse`, `assertEqual`, `assertAlmostEqual`, `assertIs`, `assertIn`,
`assertListEqual`, `assertIsInstance`, callable/context-manager
`assertRaises`/`assertRaisesRegex`, fail-fast `subTest` contexts, and module-level
`main()` discovery (including inherited test methods). Assertion failures and
unexpected exceptions propagate to the runner. This is not full CPython
`unittest`: it has no result aggregation, CLI options, skips, or test suites;
`main()` returns the number of cases on success instead of raising SystemExit.
Regex assertions currently accept Go/RE2 syntax, not all Python `re` patterns;
unsupported patterns fail explicitly rather than being silently ignored.
