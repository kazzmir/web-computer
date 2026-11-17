import { pythonTokens } from './python-tokens.js';

export function indexBrackets(text) {
  const pairs = new Map();
  const offsets = new Set();
  const closing = { ')': '(', ']': '[', '}': '{' };
  let stack = [];
  for (const token of pythonTokens(text)) {
    if (token.kind !== 'punctuation' || !'()[]{}'.includes(token.value)) continue;
    offsets.add(token.start);
    if ('([{'.includes(token.value)) stack.push(token);
    else if (stack.at(-1)?.value === closing[token.value]) {
      const open = stack.pop();
      pairs.set(open.start, token.start);
      pairs.set(token.start, open.start);
    } else stack = []; // Never pair across an incorrectly nested delimiter.
  }
  const lines = text.split('\n');
  let offset = 0;
  const lineStarts = lines.map((line) => { const start = offset; offset += line.length + 1; return start; });
  return { text, pairs, offsets, lines, lineStarts };
}

export function matchingBrackets(index, start, end = start) {
  if (start !== end) return null;
  // Prefer the character under the caret, then the one immediately before it.
  const at = index.offsets.has(start) ? start : start > 0 && index.offsets.has(start - 1) ? start - 1 : null;
  return at !== null && index.pairs.has(at) ? [at, index.pairs.get(at)] : null;
}
