import assert from 'node:assert/strict';
import { indexFunctions, functionAtOffset } from './function-index.js';

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
// Definition metadata must not overwrite a reference token's own start. Doing
// so makes a call's hover range cover unrelated code in the function body.
const pygameSource = 'import pygame as pg\ndef main():\n    """Run the game."""\n    for event in pg.event.get():\n        print(event)\nmain()\n';
const pygameIndex = indexFunctions(pygameSource);
for (const token of pygameIndex.known) {
  assert.equal(pygameSource.slice(token.start, token.end), token.value, 'each hover range must cover exactly its identifier');
}
const mainTokens = pygameIndex.known.filter((token) => token.value === 'main');
assert.deepEqual(mainTokens.map((token) => token.start), [pygameSource.indexOf('main'), pygameSource.lastIndexOf('main')]);
for (const name of ['pg.event.get', 'event.get', 'get']) {
  const offset = pygameSource.indexOf(name);
  assert.equal(pygameIndex.known.some((token) => token.start <= offset && offset < token.end), false, `${name} must not inherit main's documentation`);
}
for (const text of [
  'def f(value):\n    """Docs for f."""\n    return value\ndef g():\n    """Docs for g."""\n    f(3)\ng()\n',
  'def g():\n    """Docs for g."""\n    f(3)\ndef f(value):\n    """Docs for f."""\n    return value\ng()\n',
]) {
  const index = indexFunctions(text);
  const call = text.indexOf('f(3)');
  const info = functionAtOffset(index, call);
  assert.equal(info?.signature, 'f(value)', 'hovering f() inside g() must document f(), even when declared later');
  assert.equal(info.documentation, 'Docs for f.');
  assert.equal(functionAtOffset(index, call + 1), null, 'parentheses are not function identifiers');
  assert.equal(functionAtOffset(index, text.indexOf('Docs for g.')), null, 'docstrings are not function identifiers');
}
// Even malformed or overlapping documentation metadata cannot supply the
// enclosing function's docs in place of the identifier under the pointer.
const bogusIndex = indexFunctions('def g():\n    unknown()\ng()');
const unknownOffset = bogusIndex.text.indexOf('unknown');
const enclosing = bogusIndex.known.find((token) => token.value === 'g');
bogusIndex.knownByStart.set(unknownOffset, { ...enclosing, start: unknownOffset, end: unknownOffset + 'unknown'.length });
assert.equal(functionAtOffset(bogusIndex, unknownOffset), null, 'require the identifier name to match, not just its range');
console.log('Function documentation index tests passed');
