import assert from 'node:assert/strict';
import { dedentLines } from './editor-input.js';

function apply(text, start, end = start) {
  const edit = dedentLines(text, start, end);
  if (!edit) return { text, start, end };
  return { text: text.slice(0, edit.start) + edit.text + text.slice(edit.end), start: edit.selectionStart, end: edit.selectionEnd };
}
assert.deepEqual(apply('        x = 1', 13), { text: '    x = 1', start: 9, end: 9 });
assert.deepEqual(apply('        x', 3), { text: '    x', start: 0, end: 0 });
assert.deepEqual(apply('  x', 3), { text: 'x', start: 1, end: 1 });
assert.deepEqual(apply('\t\tx', 3), { text: '\tx', start: 2, end: 2 });
assert.deepEqual(apply('  \tx', 4), { text: 'x', start: 1, end: 1 });
assert.deepEqual(apply('    a\n        b\n    c', 0, 16), { text: 'a\n    b\n    c', start: 0, end: 8 });
assert.deepEqual(apply('\n    x', 0), { text: '\n    x', start: 0, end: 0 });
assert.equal(dedentLines('x', 0, 1), null);
assert.equal(dedentLines('', 0, 0), null);
console.log('Dedent tests passed');
