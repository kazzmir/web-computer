import assert from 'node:assert/strict';
import { indexFunctions } from './function-index.js';

const docsFor = (source, name) => indexFunctions(source).known.filter((token) => token.value === name);
assert(docsFor('print(len([1, 2]))', 'print').length === 1);
assert(docsFor('print(len([1, 2]))', 'len').length === 1);
assert.equal(docsFor('# print(1)\ntext = "print(1)"\nobj.print(1)', 'print').length, 0);
assert.equal(docsFor('unknown(3)\nobj.unknown()', 'unknown').length, 0);
assert.equal(docsFor('call(print=3)', 'print').length, 0);
for (const shadow of [
  'print = 3', 'print += 1', 'print: int', 'a, print = values', '(a, print) = values',
  'for print in values:\n    pass', '[print for print in values]',
  'def f(print):\n    pass', 'f = lambda print: print',
  'import print', 'import other as print', 'from other import print',
  'with context as print:\n    pass', 'try:\n    pass\nexcept Exception as print:\n    pass',
  'del print', '(print := 3)', 'class print:\n    pass',
  'match x:\n    case print:\n        pass',
  '@decorator\ndef print(x):\n    pass', 'if condition:\n    def print(x):\n        pass',
]) {
  assert.equal(docsFor(shadow + '\nprint(1)', 'print').length, 0, `must not guess shadowed print: ${shadow}`);
}
for (const dynamic of ['from other import *', 'exec(code)', 'eval(code)', 'globals()["print"] = other', '__builtins__ = other']) {
  assert.equal(docsFor(dynamic + '\nprint(1)', 'print').length, 0);
}
const source = `def greet(name, count=2):
    """Greet a person.

    Args:
        name: Person to greet.
    """
    return name * count

greet("World")
`;
const known = docsFor(source, 'greet');
assert.equal(known.length, 2);
assert.equal(known[1].signature, 'greet(name, count=2)');
assert.equal(known[1].documentation, 'Greet a person.\n\nArgs:\n    name: Person to greet.');
assert.equal(docsFor(source + '\ngreet = 7', 'greet').length, 0);
assert.equal(docsFor(source + '\ndef greet(x): pass', 'greet').length, 0);
assert.equal(docsFor('greet(1)\n' + source, 'greet').some((token) => token.start === 0), false, 'do not document an unresolved forward reference');
assert.equal(docsFor('@decorator\n' + source, 'greet').length, 0);
assert.equal(docsFor('def f():\n    def nested(x):\n        pass\n    nested(1)', 'nested').length, 0);
assert.equal(docsFor('async def work(\n    value: int = 3,\n) -> int:\n    """Return a number."""\n    return value\nwork()', 'work')[1].documentation, 'Return a number.');
assert.equal(docsFor('def f(): "Short doc."\nf()', 'f')[1].documentation, 'Short doc.');
assert.equal(docsFor('def f():\n    return 3\nf()', 'f')[1].documentation, 'Function defined in this file. No docstring provided.');
assert.equal(docsFor('def f():\n    """print <img src=x>"""\nf()', 'print').length, 0, 'do not inspect docstring contents as code');
console.log('Function documentation index tests passed');
