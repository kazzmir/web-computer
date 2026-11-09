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
| `test_xml.py` | 2 | XML 1.0 name/text validation, Unicode boundaries, forbidden characters and surrogates |
| `test_augassign.py` | 7 | All augmented arithmetic operators, custom/reflected/in-place slots, matrix operators, sequence mutation, numeric dictionary keys |
| `test_errno.py` | 3 | Error constants, numeric reverse mapping, module-attribute consistency |
| `test_print.py` | 9 | Output redirection, separators/endings, flush behavior/errors, stdout replacement, Python 2 migration hints |
| `test_subclassinit.py` | 17 | PEP 487 subclass/descriptor hooks, cooperative MRO, class keywords, metaclass construction, descriptor error notes |
| `test_decorators.py` | 16 | Method-wrapper metadata, arbitrary decorator expressions, ordering, memoization, function attributes, class decorators |
| `test_richcmp.py` | 11 | All rich comparisons, arbitrary result objects, reflected methods, lexicographic lists, recursion errors and diagnostic messages |

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
The `xml` module supplies name/text validation only; DOM, SAX, etree, and XML
parsers are not implemented. Python string code-point operations preserve lone
surrogates; UTF-8 encoding rejects them unless an explicit error handler permits
them.

Matrix `@`/`@=` implements the special-method protocol; it does not implicitly
add matrix operations to built-in sequences. Value-dictionary literals,
comprehensions, and `dict(...)` use a generic-key representation when necessary.
Empty `{}` and constructor-created dictionaries now accept later non-string
keys without breaking aliases (including tuple-key memoization). String-key
storage is shared with native namespace APIs so `exec`/`eval` mutations remain
visible. Immutable built-in keys use Python equality; custom hashable objects,
NaN keys, and full insertion-order guarantees are not implemented. Native
namespace maps and nonempty string-only literals still use the string-only
fast path and cannot later accept non-string keys through aliased mutation.

Decorators accept general expressions, including assignment expressions.
Assignment expressions currently support name targets in parentheses,
decorators, and `if`/`elif`/`while` conditions; comprehension binding rules are
not implemented and are explicitly rejected. `str.format` supports automatic,
numbered and named fields, attribute/index access, escaped braces, conversions,
and nested format specifications, sharing the existing value-formatting engine.

`errno` exposes a common POSIX subset using Go's target-platform `syscall`
values (including Go's js/wasm table in browsers), not an exhaustive copy of
all host C-library constants. Each context receives its own mutable `errorcode`
dictionary; aliases map to one deterministic representative name.

`io` currently supplies a limited `StringIO` text stream with write/read,
seek/tell, getvalue, flush/close, and context management. Only the default
`newline='\\n'` mode is supported; file wrappers, binary streams, and full
`io` class hierarchies are not implemented. The minimal `test.support` module
provides real `captured_stdout`, exception-safe `swap_attr`, and `gc_collect`
(using Go's garbage collector). It is not a copy of CPython's full test-support
package and does not skip cases or suppress test failures. It also supplies
`infinite_recursion` as a real temporary VM recursion-limit context/decorator;
`no_tracing` is transparent because this runtime has no tracing facility.

Class creation now initializes class cells before descriptor/subclass hooks,
implicitly binds `__init_subclass__` as a class method, notifies descriptors from
a namespace snapshot, and preserves hook exceptions with diagnostic notes.
`super()` supports calling-frame inference and explicit class/instance binding
through the C3 MRO. This is not complete `super` descriptor emulation.
`types` provides `prepare_class` and `new_class` for ordinary type bases; PEP 560
base resolution and the rest of `types` are not provided. Native container
subclass instance storage remains limited (the `dict` subclass tests here test
class creation, not dictionary-instance behavior).

Rich comparisons preserve arbitrary return objects, honor reflected subtype
priority, propagate exceptions, and fall back to identity only after both
operands decline equality. Lists compare lexicographically and truth-test item
comparison results. `bool(...)` now follows the truth protocol, validating
`__bool__` results and rejecting negative lengths. `operator` supplies the six
comparison functions/dunder aliases and `not_`; `collections` supplies a limited
`UserList` wrapper, not its full container/ABC API. `random.shuffle` performs
in-place Fisher-Yates shuffling using the context's generator.

`sys.getrecursionlimit`/`setrecursionlimit` control per-context Python VM frame
depth (default 1000), including generator frame execution. Counters unwind on
errors and limits are isolated between contexts. Native-only recursive paths
are not all covered by this frame guard; this is not complete CPython stack
accounting.

The runtime currently supplies a **limited, fail-fast `unittest` subset**:
`TestCase`, `setUp`/`tearDown`, LIFO `addCleanup`/`doCleanups`, `assertTrue`,
`assertFalse`, `assertEqual`, `assertAlmostEqual`, `assertIs`, `assertIn`,
`assertListEqual`, `assertIsInstance`, `assertHasAttr`, `assertNotIn`,
`assertCountEqual`, callable/context-manager
`assertRaises`/`assertRaisesRegex`, `assertRegex`, fail-fast `subTest` contexts, and module-level
`main()` discovery (including inherited test methods). Assertion failures and
unexpected exceptions propagate to the runner. This is not full CPython
`unittest`: it has no result aggregation, CLI options, skips, or test suites;
`main()` returns the number of cases on success instead of raising SystemExit.
Regex assertions currently accept Go/RE2 syntax, not all Python `re` patterns;
unsupported patterns fail explicitly rather than being silently ignored.
