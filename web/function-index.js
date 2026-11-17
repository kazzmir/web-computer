import { pythonTokens } from './python-tokens.js';
import { builtinFunctionDocs } from './function-docs.js';

function statements(text, tokens) {
  const result = [];
  let current = [];
  let depth = 0;
  let previous = 0;
  const finish = () => {
    if (!current.length) return;
    const start = current[0].start;
    const lineStart = text.lastIndexOf('\n', start - 1) + 1;
    result.push({ tokens: current, topLevel: text.slice(lineStart, start) === '' });
    current = [];
  };
  for (const token of tokens) {
    const gap = text.slice(previous, token.start);
    if (depth === 0 && gap.includes('\n') && !/\\\r?\n\s*$/.test(gap)) finish();
    if (token.value === ';' && depth === 0) { finish(); previous = token.end; continue; }
    current.push(token);
    if ('([{'.includes(token.value)) depth++;
    if (')]}'.includes(token.value)) depth = Math.max(0, depth - 1);
    previous = token.end;
  }
  finish();
  return result;
}

function stringValue(token) {
  const match = /^([rRuUbBfF]{0,2})("""|'''|"|')/.exec(token);
  if (!match || /[bf]/i.test(match[1]) || !token.endsWith(match[2])) return null;
  let body = token.slice(match[0].length, -match[2].length);
  if (!/r/i.test(match[1])) {
    const escapes = { n: '\n', r: '\r', t: '\t', b: '\b', f: '\f', v: '\v', a: '\x07', '\\': '\\', '"': '"', "'": "'" };
    body = body.replace(/\\(?:\r?\n|x[\da-fA-F]{2}|u[\da-fA-F]{4}|U[\da-fA-F]{8}|.)/g, (value) => {
      const escape = value.slice(1);
      if (escape.includes('\n')) return '';
      if (escape in escapes) return escapes[escape];
      if (/^[xuU]/.test(escape)) {
        const point = parseInt(escape.slice(1), 16);
        if (point <= 0x10ffff) return String.fromCodePoint(point);
      }
      return value;
    });
  }
  const lines = body.replace(/\r\n/g, '\n').replace(/\t/g, '    ');
  const parts = lines.split('\n');
  const indents = parts.slice(1).filter((line) => line.trim()).map((line) => line.match(/^ */)[0].length);
  const indent = indents.length ? Math.min(...indents) : 0;
  const clean = [parts[0].trimStart(), ...parts.slice(1).map((line) => line.slice(indent))];
  while (clean.length && !clean[0].trim()) clean.shift();
  while (clean.length && !clean.at(-1).trim()) clean.pop();
  return clean.join('\n').slice(0, 6000);
}

// Deliberately conservative static binding analysis, not a Python evaluator.
// Any ambiguous binding anywhere suppresses documentation for that name.
export function indexFunctions(text) {
  const tokens = [...pythonTokens(text)].filter((token) => token.kind !== 'comment');
  const groups = statements(text, tokens);
  const unsafe = new Set();
  const declarations = new Map();
  const definitions = new Map();
  let dynamic = false;
  const identifiers = (items) => items.filter((token) => token.kind === 'identifier');
  const invalidate = (items) => identifiers(items).forEach((token) => unsafe.add(token.value));
  const declare = (name) => declarations.set(name, (declarations.get(name) ?? 0) + 1);

  for (let groupIndex = 0; groupIndex < groups.length; groupIndex++) {
    const group = groups[groupIndex];
    let items = group.tokens;
    const defIndex = items[0]?.value === 'async' ? 1 : 0;
    if (items[defIndex]?.value === 'def' && items[defIndex + 1]?.kind === 'identifier') {
      const name = items[defIndex + 1];
      declare(name.value);
      let depth = 0;
      let closing = -1;
      if (items[defIndex + 2]?.value === '(') {
        for (let i = defIndex + 2; i < items.length; i++) {
          if (items[i].value === '(') depth++;
          if (items[i].value === ')' && --depth === 0) { closing = i; break; }
        }
      }
      const colon = closing < 0 ? -1 : items.findIndex((token, i) => i > closing && token.value === ':');
      // Parameter names, annotations, and defaults are treated conservatively;
      // no speculative scope inference for a shadowed builtin or function.
      invalidate(items.slice(defIndex + 2, colon < 0 ? undefined : colon));
      const decorated = groups[groupIndex - 1]?.tokens[0]?.value === '@';
      if (group.topLevel && !decorated && colon >= 0) {
        const body = items.slice(colon + 1);
        const next = groups[groupIndex + 1];
        const docTokens = body.length ? body : next && !next.topLevel ? next.tokens : [];
        const doc = docTokens.length === 1 && docTokens[0].kind === 'string' ? stringValue(docTokens[0].value) : null;
        definitions.set(name.value, {
          signature: text.slice(name.start, items[colon].start).trim().replace(/\n\s*/g, ' ').slice(0, 2000),
          documentation: doc || 'Function defined in this file. No docstring provided.',
          origin: 'This file',
          definitionStart: name.start,
        });
      } else unsafe.add(name.value);
      items = colon < 0 ? [] : items.slice(colon + 1);
    } else if (items[0]?.value === 'class') {
      if (items[1]) { declare(items[1].value); unsafe.add(items[1].value); }
    }

    if (items[0]?.value === 'import' || items[0]?.value === 'from') {
      invalidate(items);
      if (items.some((token) => token.value === '*')) dynamic = true;
    }
    if (['del', 'global', 'nonlocal', 'case'].includes(items[0]?.value)) invalidate(items.slice(1));
    if (items[0]?.kind === 'identifier' && items[1]?.value === ':') unsafe.add(items[0].value);
    let depth = 0;
    for (let i = 0; i < items.length; i++) {
      const token = items[i];
      if (token.value === '__builtins__' || (['exec', 'eval', 'globals', 'locals'].includes(token.value) && items[i + 1]?.value === '(')) dynamic = true;
      if (token.value === 'as') invalidate(items.slice(i + 1));
      if (token.value === 'for') {
        const end = items.findIndex((next, j) => j > i && next.value === 'in');
        if (end >= 0) invalidate(items.slice(i + 1, end));
      }
      if (token.value === 'lambda') {
        const end = items.findIndex((next, j) => j > i && next.value === ':');
        if (end >= 0) invalidate(items.slice(i + 1, end));
      }
      if (token.value === ':=' && items[i - 1]) unsafe.add(items[i - 1].value);
      if (depth === 0 && /^(?:=|\+=|-=|\*=|\/=|\/\/=|%=|&=|\|=|\^=|>>=|<<=|\*\*=)$/.test(token.value)) invalidate(items.slice(0, i));
      if ('([{'.includes(token.value)) depth++;
      if (')]}'.includes(token.value)) depth = Math.max(0, depth - 1);
    }
  }

  const functions = new Map();
  if (!dynamic) {
    for (const [name, [signature, documentation]] of builtinFunctionDocs) {
      if (!unsafe.has(name) && !declarations.has(name)) functions.set(name, { signature, documentation, origin: 'Python built-in' });
    }
    for (const [name, info] of definitions) {
      if (!unsafe.has(name) && declarations.get(name) === 1) functions.set(name, info);
    }
  }
  const known = tokens.filter((token, i) => {
    const info = functions.get(token.value);
    return token.kind === 'identifier' && info && tokens[i - 1]?.value !== '.' && tokens[i + 1]?.value !== '=' && (info.definitionStart === undefined || token.start >= info.definitionStart);
  }).map((token) => ({ ...token, ...functions.get(token.value) }));
  const lines = text.split('\n');
  let offset = 0;
  const lineStarts = lines.map((line) => { const start = offset; offset += line.length + 1; return start; });
  return { text, known, lines, lineStarts };
}
