import assert from 'node:assert/strict';
import { indexBrackets, matchingBrackets } from './bracket-pairs.js';

for (const text of ['()', '[]', '{}', 'f([{"key": (1, 2)}])', 'x = (\n    [1, {2: 3}]\n)']) {
  const index = indexBrackets(text);
  for (const [from, to] of index.pairs) {
    assert.deepEqual(matchingBrackets(index, from), [from, to]);
    assert.equal(index.pairs.get(to), from);
  }
  assert(index.pairs.size > 0);
}
assert.deepEqual(matchingBrackets(indexBrackets('(abc)'), 5), [4, 0], 'caret after a closing bracket should find the opener at offset zero');
assert.equal(matchingBrackets(indexBrackets('(abc)'), 2), null);
assert.equal(matchingBrackets(indexBrackets('(abc)'), 0, 5), null, 'nonempty selections do not match');
for (const text of ['(', ']', '([)]', '(]']) assert.equal(indexBrackets(text).pairs.size, 0, `must not pair malformed nesting ${text}`);
for (const text of ['"([{}])"', "'([{}])'", '# ([{}])', '"""(\n[{}]\n)"""', 'r"(abc)"', 'f"{call()}"']) {
  const index = indexBrackets(text);
  assert.equal(index.pairs.size, 0, `brackets in strings/comments should be ignored: ${text}`);
  assert.equal(index.offsets.size, 0);
}
const comments = 'f(  # stray ) ] }\n    "escaped \\\" )", [1]\n)';
const index = indexBrackets(comments);
assert.equal(index.pairs.get(comments.indexOf('(')), comments.lastIndexOf(')'));
assert.equal(index.pairs.get(comments.indexOf('[1]')), comments.indexOf('[1]') + 2);
console.log('Bracket matching index tests passed');
