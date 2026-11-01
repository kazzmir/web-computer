# CPython regression tests

These test files are copied unchanged from
`/home/ubuntu/tmp/Python-3.15.2/Lib/test`:

| File | Cases | Coverage |
| --- | ---: | --- |
| `test_unary.py` | 6 | Unary arithmetic, complex numbers, big integers, precedence, invalid operands |

The upstream license is retained in `LICENSE.cpython`.

Run the tests against this repository's Python runtime (not the host Python):

```sh
go test ./test -v
go run ./gpython test/test_unary.py
```

`runtime_test.go` checks exact case counts to guard against empty discovery.
Each file runs in a fresh interpreter context. No cases are skipped or weakened.

The runtime currently supplies a **limited, fail-fast `unittest` subset**:
`TestCase`, `setUp`/`tearDown`, `assertTrue`, `assertFalse`, `assertEqual`,
`assertIsInstance`, callable/context-manager `assertRaises`, and module-level
`main()` discovery (including inherited test methods). Assertion failures and
unexpected exceptions propagate to the runner. This is not full CPython
`unittest`: it has no result aggregation, CLI options, skips, or test suites;
`main()` returns the number of cases on success instead of raising SystemExit.
